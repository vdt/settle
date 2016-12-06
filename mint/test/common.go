package test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/lib/token"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/app"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/mint/model"
	goji "goji.io"
)

const (
	// PostLatency is the expected latency between a test running an the created
	// stamp of an object we created within a test.
	PostLatency time.Duration = 500 * time.Millisecond
)

var userIdx int

func init() {
	userIdx = 0
}

// Mint represents a test mint.
type Mint struct {
	Server  *httptest.Server
	Mux     *goji.Mux
	Env     *env.Env
	DB      *sqlx.DB
	Ctx     context.Context
	TmpFile string
}

// CreateMint creates a new test mint with an in-memory DB and returns
// test.Mint object.
func CreateMint(
	t *testing.T,
) *Mint {
	ctx := context.Background()

	mintEnv := env.Env{
		Environment: env.QA,
		Config:      map[env.ConfigKey]string{},
	}
	ctx = env.With(ctx, &mintEnv)

	tmpFile :=
		filepath.Join(os.TempDir(), token.New("test")+".db")

	mintDB, err := db.NewSqlite3DBForPath(ctx, tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	err = model.CreateMintDBTables(ctx, mintDB)
	if err != nil {
		t.Fatal(err)
	}
	ctx = db.WithDB(ctx, mintDB)

	mux := goji.NewMux()
	mux.Use(requestlogger.Middleware)
	mux.Use(recoverer.Middleware)
	mux.Use(db.Middleware(db.GetDB(ctx)))
	mux.Use(env.Middleware(env.Get(ctx)))
	mux.Use(authentication.Middleware)

	(&app.Controller{}).Bind(mux)

	// We don't start an async worker in tests and rely on manually running
	// tasks when needed instead.

	m := Mint{
		Server:  httptest.NewServer(mux),
		Mux:     mux,
		Env:     &mintEnv,
		DB:      mintDB,
		Ctx:     ctx,
		TmpFile: tmpFile,
	}
	m.Env.Config[mint.EnvCfgMintHost] = m.Server.URL[7:]

	logging.Logf(ctx, "Creating test mint: minst_host=%s",
		m.Env.Config[mint.EnvCfgMintHost])

	return &m
}

// Close closes the mint after usage.
func (m *Mint) Close() {
	defer os.Remove(m.TmpFile)
	m.DB.Close()
}

// MintUser reprensents a user of a mint, generally generated by CreateUser.
type MintUser struct {
	Mint     *Mint
	Username string
	Password string
	Address  string
}

var userFirstnames = []string{
	"kurt", "alan", "albert", "john", "henri", "charles", "isaac", "louis",
	"niels", "alexander", "thomas", "max", "rosalind",
}

// CreateUser creates a user and generates an associated MintUser
func (m *Mint) CreateUser(
	t *testing.T,
) *MintUser {
	userIdx++
	username := token.New(userFirstnames[userIdx%len(userFirstnames)])
	password := token.New("password")

	_, err := model.CreateUser(m.Ctx, username, password)
	if err != nil {
		t.Fatal(err)
	}
	m.Env.Config[mint.EnvCfgMintHost] = m.Server.URL[7:]

	logging.Logf(m.Ctx, "Creating test mint: minst_host=%s",
		m.Env.Config[mint.EnvCfgMintHost])

	return &MintUser{
		m, username, password,
		fmt.Sprintf("%s@%s", username, m.Env.Config[mint.EnvCfgMintHost]),
	}
}

// Post posts to a specified endpoint on the mint.
func (m *Mint) Post(
	t *testing.T,
	user *MintUser,
	path string,
	params url.Values,
) (int, svc.Resp) {
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s%s", m.Server.URL, path),
		strings.NewReader(params.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if user != nil {
		req.SetBasicAuth(user.Username, user.Password)
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}

	return r.StatusCode, raw
}

// Post posts to a specified endpoint on the mint.
func (u *MintUser) Post(
	t *testing.T,
	path string,
	params url.Values,
) (int, svc.Resp) {
	return u.Mint.Post(t, u, path, params)
}

// Get gets a specified endpoint on the mint.
func (m *Mint) Get(
	t *testing.T,
	user *MintUser,
	path string,
) (int, svc.Resp) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s%s", m.Server.URL, path), nil)
	if err != nil {
		t.Fatal(err)
	}
	if user != nil {
		req.SetBasicAuth(user.Username, user.Password)
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}

	return r.StatusCode, raw
}

// Get gets a specified endpoint on the mint.
func (u *MintUser) Get(
	t *testing.T,
	path string,
) (int, svc.Resp) {
	return u.Mint.Get(t, u, path)
}

// CreateAsset creates a new assset for this test user
func (u *MintUser) CreateAsset(
	t *testing.T,
	code string,
	scale int8,
) mint.AssetResource {
	_, raw := u.Post(t,
		"/assets",
		url.Values{
			"code":  {code},
			"scale": {fmt.Sprintf("%d", scale)},
		})
	var asset mint.AssetResource
	if err := raw.Extract("asset", &asset); err != nil {
		t.Fatal(err)
	}

	return asset
}

// CreateOffer creates a new ofer for this test user
func (u *MintUser) CreateOffer(
	t *testing.T,
	pair string,
	price string,
	amount *big.Int,
) mint.OfferResource {
	_, raw := u.Post(t,
		"/offers",
		url.Values{
			"pair":   {pair},
			"price":  {price},
			"amount": {amount.String()},
		})
	var offer mint.OfferResource
	if err := raw.Extract("offer", &offer); err != nil {
		t.Fatal(err)
	}

	return offer
}
