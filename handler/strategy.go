package handler

import (
	"encoding/json"
	"log"
	"main/data"
	"main/strategies"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
)

var strategyList = [1]strategies.StrategyInfo{
	strategies.AcceleratingDualMomentumInfo(),
}

var strategyMap = make(map[string]strategies.StrategyInfo)

// IntializeStrategyMap configure the strategy map
func IntializeStrategyMap() {
	for ii := range strategyList {
		strat := strategyList[ii]
		strategyMap[strat.Shortcode] = strat
	}
}

// ListStrategies get a list of all strategies
func ListStrategies(c *fiber.Ctx) error {
	return c.JSON(strategyList)
}

// GetStrategy get configuration for a specific strategy
func GetStrategy(c *fiber.Ctx) error {
	shortcode := c.Params("id")
	if strategy, ok := strategyMap[shortcode]; ok {
		return c.JSON(strategy)
	}
	return fiber.ErrNotFound
}

// RunStrategy execute strategy
func RunStrategy(c *fiber.Ctx) error {
	shortcode := c.Params("id")

	if strat, ok := strategyMap[shortcode]; ok {
		credentials := make(map[string]string)

		// get tiingo token from jwt claims
		user := c.Locals("user").(*jwt.Token)
		claims := user.Claims.(jwt.MapClaims)
		tiingoToken := claims["https://pennyvault.com/tiingo_token"].(string)

		credentials["tiingo"] = tiingoToken
		manager := data.NewManager(credentials)

		params := map[string]json.RawMessage{}
		if err := json.Unmarshal(c.Body(), &params); err != nil {
			log.Println(err)
			return fiber.ErrBadRequest
		}

		stratObject, err := strat.Factory(params)
		if err != nil {
			log.Println(err)
			return fiber.ErrBadRequest
		}

		performance, err := stratObject.Compute(manager)
		if err != nil {
			log.Println(err)
			return fiber.ErrBadRequest
		}

		return c.JSON(performance)
	}

	return fiber.ErrNotFound
}
