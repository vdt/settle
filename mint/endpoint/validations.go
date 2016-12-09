package endpoint

import (
	"context"
	"math/big"
	"regexp"
	"strconv"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/model"
)

// PriceRegexp is used to validate and parse a transaction price.
var PriceRegexp = regexp.MustCompile(
	"^([0-9]+)\\/([0-9]+)$")

// ValidatePrice validates a price (pB/pQ).
func ValidatePrice(
	ctx context.Context,
	price string,
) (*big.Int, *big.Int, error) {
	m := PriceRegexp.FindStringSubmatch(price)
	if len(m) == 0 {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			400, "price_invalid",
			"The offer price you provided is invalid: %s. Prices must have "+
				"the form 'pB/pQ' where pB is the base asset price and pQ "+
				"is the quote asset price.",
			price,
		))
	}
	var basePrice big.Int
	_, success := basePrice.SetString(m[1], 10)
	if !success ||
		basePrice.Cmp(new(big.Int)) < 0 ||
		basePrice.Cmp(model.MaxAssetAmount) >= 0 {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			400, "price_invalid",
			"The base asset price you provided is invalid: %s. Asset prices "+
				"must be integers between 0 and 2^128.",
			m[1],
		))
	}

	var quotePrice big.Int
	_, success = quotePrice.SetString(m[2], 10)
	if !success ||
		quotePrice.Cmp(new(big.Int)) < 0 ||
		quotePrice.Cmp(model.MaxAssetAmount) >= 0 {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			400, "price_invalid",
			"The quote asset price you provided is invalid: %s. Asset prices "+
				"must be integers between 0 and 2^128.",
			m[2],
		))
	}

	return &basePrice, &quotePrice, nil
}

// ValidateAmount validates the amount of an asset.
func ValidateAmount(
	ctx context.Context,
	amount string,
) (*big.Int, error) {
	var a big.Int
	_, success := a.SetString(amount, 10)
	if !success ||
		a.Cmp(new(big.Int)) < 0 ||
		a.Cmp(model.MaxAssetAmount) >= 0 {
		return nil, errors.Trace(errors.NewUserErrorf(nil,
			400, "amount_invalid",
			"The amount you provided is invalid: %s. Amounts must be "+
				"integers between 0 and 2^128.",
			amount,
		))
	}

	return &a, nil
}

// ValidateAssetPair validates an asset pair.
func ValidateAssetPair(
	ctx context.Context,
	pair string,
) ([]mint.AssetResource, error) {
	p, err := mint.AssetResourcesFromPair(ctx, pair)
	if err != nil {
		return nil, errors.Trace(errors.NewUserErrorf(err,
			400, "pair_invalid",
			"The asset pair you provided is invalid: %s.",
			pair,
		))
	}

	return p, nil
}

// ValidatePath validates a path of offers.
func ValidatePath(
	ctx context.Context,
	path []string,
) ([]string, error) {
	for _, offer := range path {
		_, _, err := mint.NormalizedOwnerAndTokenFromID(ctx, offer)
		if err != nil {
			return nil, errors.Trace(errors.NewUserErrorf(err,
				400, "path_invalid",
				"The offer id you provided in `path[]` is invalid: %s. Offer ids "+
					"must have the form kgodel@princetown.edu[offer_*]",
				offer,
			))
		}
	}

	return path, nil
}

// ValidateID validates the ID of an object
func ValidateID(
	ctx context.Context,
	id string,
) (*string, *string, *string, error) {
	owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, id)
	if err != nil {
		return nil, nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			400, "id_invalid",
			"The id you provided is invalid: %s. Ids must have the form "+
				"kgodel@princetown.edu[xxxx_*].",
			id,
		))
	}

	return &id, &owner, &token, nil
}

// ValidateSecret validates a secret.
func ValidateSecret(
	ctx context.Context,
	secret string,
) (*string, error) {
	if len(secret) != 16 {
		return nil, errors.Trace(errors.NewUserErrorf(nil,
			400, "secret_invalid",
			"The secret you provided is structurally invalid: %s.",
			secret,
		))
	}

	return &secret, nil
}

// ValidateHop validates a hop.
func ValidateHop(
	ctx context.Context,
	hop string,
) (*int8, error) {
	h, err := strconv.ParseInt(hop, 10, 8)
	if err != nil || h < 0 {
		return nil, errors.Trace(errors.NewUserErrorf(err,
			400, "hop_invalid",
			"The transaction hop provided is invalid: %s. Transaction "+
				"hops must be 8 bits positive integers.",
			hop,
		))
	}
	converted := int8(h)

	return &converted, nil
}