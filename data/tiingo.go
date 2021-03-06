package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	dataframe "github.com/rocketlaunchr/dataframe-go"
	imports "github.com/rocketlaunchr/dataframe-go/imports"
	log "github.com/sirupsen/logrus"
)

type tiingo struct {
	apikey string
}

type tiingoJSONResponse struct {
	Date        string  `json:"date"`
	Close       float64 `json:"close"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Open        float64 `json:"open"`
	Volume      int64   `json:"volume"`
	AdjClose    float64 `json:"adjClose"`
	AdjHigh     float64 `json:"adjHigh"`
	AdjLow      float64 `json:"adjLow"`
	AdjOpen     float64 `json:"adjOpen"`
	AdjVolume   int64   `json:"adjVolume"`
	DivCash     float64 `json:"divCash"`
	SplitFactor float64 `json:"splitFactor"`
}

var tiingoTickersURL = "https://apimedia.tiingo.com/docs/tiingo/daily/supported_tickers.zip"
var tiingoAPI = "https://api.tiingo.com"

// NewTiingo Create a new Tiingo data provider
func NewTiingo(key string) tiingo {
	return tiingo{
		apikey: key,
	}
}

// Date provider functions

// LastTradingDayOfWeek return the last trading day of the week
func (t tiingo) LastTradingDay(forDate time.Time, frequency string) (time.Time, error) {
	symbol := "SPY"
	url := fmt.Sprintf("%s/tiingo/daily/%s/prices?startDate=%s&endDate=%s&resampleFreq=%s&token=%s", tiingoAPI, symbol, forDate.Format("2006-01-02"), forDate.Format("2006-01-02"), frequency, t.apikey)

	resp, err := http.Get(url)
	if err != nil {
		log.WithFields(log.Fields{
			"Function":  "data/tiingo.go:LastTradingDay",
			"ForDate":   forDate,
			"Frequency": frequency,
			"Error":     err,
		}).Error("HTTP error response")
		return time.Time{}, err
	}

	if resp.StatusCode >= 400 {
		log.WithFields(log.Fields{
			"Function":   "data/tiingo.go:LastTradingDay",
			"ForDate":    forDate,
			"Frequency":  frequency,
			"StatusCode": resp.StatusCode,
			"Error":      err,
		}).Error("HTTP error response")
		return time.Time{}, fmt.Errorf("HTTP request returned invalid status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"Function":  "data/tiingo.go:LastTradingDay",
			"ForDate":   forDate,
			"Frequency": frequency,
			"Body":      string(body),
			"Error":     err,
		}).Error("Failed to read HTTP body")
		return time.Time{}, err
	}

	jsonResp := []tiingoJSONResponse{}
	err = json.Unmarshal(body, &jsonResp)
	if err != nil {
		log.WithFields(log.Fields{
			"Function":  "data/tiingo.go:LastTradingDay",
			"ForDate":   forDate,
			"Frequency": frequency,
			"Body":      string(body),
			"Error":     err,
		}).Error("Failed to parse JSON")
		return time.Time{}, err
	}

	if len(jsonResp) > 0 {
		dtParts := strings.Split(jsonResp[0].Date, "T")
		if len(dtParts) == 0 {
			log.WithFields(log.Fields{
				"Function":  "data/tiingo.go:LastTradingDay",
				"ForDate":   forDate,
				"Frequency": frequency,
				"DateStr":   jsonResp[0].Date,
				"Error":     err,
			}).Error("Invalid date format")
			return time.Time{}, errors.New("Invalid date format")
		}
		lastDay, err := time.Parse("2006-01-02", dtParts[0])
		if err != nil {
			log.WithFields(log.Fields{
				"Function":   "data/tiingo.go:LastTradingDay",
				"ForDate":    forDate,
				"Frequency":  frequency,
				"StatusCode": resp.StatusCode,
				"Error":      err,
			}).Error("Cannot parse date")
			return time.Time{}, err
		}

		return lastDay, nil
	}

	return time.Time{}, errors.New("No data returned")
}

// LastTradingDayOfWeek return the last trading day of the week
func (t tiingo) LastTradingDayOfWeek(forDate time.Time) (time.Time, error) {
	return t.LastTradingDay(forDate, "weekly")
}

// LastTradingDayOfMonth return the last trading day of the month
func (t tiingo) LastTradingDayOfMonth(forDate time.Time) (time.Time, error) {
	return t.LastTradingDay(forDate, "monthly")
}

// LastTradingDayOfYear return the last trading day of the year
func (t tiingo) LastTradingDayOfYear(forDate time.Time) (time.Time, error) {
	return t.LastTradingDay(forDate, "annually")
}

// Provider functions

func (t tiingo) DataType() string {
	return "security"
}

func (t tiingo) GetDataForPeriod(symbol string, metric string, frequency string, begin time.Time, end time.Time) (data *dataframe.DataFrame, err error) {
	validFrequencies := map[string]bool{
		FrequencyDaily:   true,
		FrequencyWeekly:  true,
		FrequencyMonthly: true,
		FrequencyAnnualy: true,
	}

	if _, ok := validFrequencies[frequency]; !ok {
		log.WithFields(log.Fields{
			"Frequency": frequency,
			"Symbol":    symbol,
			"Metric":    metric,
			"StartTime": begin.String(),
			"EndTime":   end.String(),
		}).Debug("Invalid frequency provided")
		return nil, fmt.Errorf("invalid frequency '%s'", frequency)
	}

	// build URL to get data
	var url string
	nullTime := time.Time{}
	if begin == nullTime || end == nullTime {
		url = fmt.Sprintf("%s/tiingo/daily/%s/prices?format=csv&resampleFreq=%s&token=%s", tiingoAPI, symbol, frequency, t.apikey)
	} else {
		url = fmt.Sprintf("%s/tiingo/daily/%s/prices?startDate=%s&endDate=%s&format=csv&resampleFreq=%s&token=%s", tiingoAPI, symbol, begin.Format("2006-01-02"), end.Format("2006-01-02"), frequency, t.apikey)
	}

	resp, err := http.Get(url)

	if err != nil {
		log.WithFields(log.Fields{
			"Url":       url,
			"Symbol":    symbol,
			"Metric":    metric,
			"Frequency": frequency,
			"StartTime": begin.String(),
			"EndTime":   end.String(),
			"Error":     err,
		}).Debug("Failed to load eod prices")
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"Url":        url,
			"Symbol":     symbol,
			"Metric":     metric,
			"Frequency":  frequency,
			"StartTime":  begin.String(),
			"EndTime":    end.String(),
			"Error":      err,
			"StatusCode": resp.StatusCode,
		}).Debug("Failed to load eod prices -- reading body failed")
		return nil, err
	}

	if resp.StatusCode >= 400 {
		log.WithFields(log.Fields{
			"Url":        url,
			"Symbol":     symbol,
			"Metric":     metric,
			"Frequency":  frequency,
			"StartTime":  begin.String(),
			"EndTime":    end.String(),
			"Body":       string(body),
			"StatusCode": resp.StatusCode,
		}).Debug("Failed to load eod prices")
		return nil, fmt.Errorf("HTTP request returned invalid status code: %d", resp.StatusCode)
	}

	floatConverter := imports.Converter{
		ConcreteType: float64(0),
		ConverterFunc: func(in interface{}) (interface{}, error) {
			v, err := strconv.ParseFloat(in.(string), 64)
			if err != nil {
				return math.NaN(), nil
			}
			return v, nil
		},
	}

	res, err := imports.LoadFromCSV(context.TODO(), bytes.NewReader(body), imports.CSVLoadOptions{
		DictateDataType: map[string]interface{}{
			"date": imports.Converter{
				ConcreteType: time.Time{},
				ConverterFunc: func(in interface{}) (interface{}, error) {
					return time.Parse("2006-01-02", in.(string))
				},
			},
			"open":      floatConverter,
			"high":      floatConverter,
			"low":       floatConverter,
			"close":     floatConverter,
			"volume":    floatConverter,
			"adjOpen":   floatConverter,
			"adjHigh":   floatConverter,
			"adjLow":    floatConverter,
			"adjClose":  floatConverter,
			"adjVolume": floatConverter,
		},
	})

	err = nil
	var timeSeries dataframe.Series
	var valueSeries dataframe.Series

	timeSeriesIdx, err := res.NameToColumn("date")
	if err != nil {
		return nil, errors.New("Cannot find time series")
	}

	timeSeries = res.Series[timeSeriesIdx]
	timeSeries.Rename(DateIdx)

	switch metric {
	case MetricOpen:
		valueSeriesIdx, err := res.NameToColumn("open")
		if err != nil {
			return nil, errors.New("open metric not found")
		}
		valueSeries = res.Series[valueSeriesIdx]
	case MetricHigh:
		valueSeriesIdx, err := res.NameToColumn("high")
		if err != nil {
			return nil, errors.New("high metric not found")
		}
		valueSeries = res.Series[valueSeriesIdx]
	case MetricLow:
		valueSeriesIdx, err := res.NameToColumn("low")
		if err != nil {
			return nil, errors.New("low metric not found")
		}
		valueSeries = res.Series[valueSeriesIdx]
	case MetricClose:
		valueSeriesIdx, err := res.NameToColumn("close")
		if err != nil {
			return nil, errors.New("close metric not found")
		}
		valueSeries = res.Series[valueSeriesIdx]
	case MetricVolume:
		valueSeriesIdx, err := res.NameToColumn("volume")
		if err != nil {
			return nil, errors.New("volume metric not found")
		}
		valueSeries = res.Series[valueSeriesIdx]
	case MetricAdjustedOpen:
		valueSeriesIdx, err := res.NameToColumn("adjOpen")
		if err != nil {
			return nil, errors.New("Adjusted open metric not found")
		}
		valueSeries = res.Series[valueSeriesIdx]
	case MetricAdjustedHigh:
		valueSeriesIdx, err := res.NameToColumn("adjHigh")
		if err != nil {
			return nil, errors.New("Adjusted high metric not found")
		}
		valueSeries = res.Series[valueSeriesIdx]
	case MetricAdjustedLow:
		valueSeriesIdx, err := res.NameToColumn("adjLow")
		if err != nil {
			return nil, errors.New("Adjusted low metric not found")
		}
		valueSeries = res.Series[valueSeriesIdx]
	case MetricAdjustedClose:
		valueSeriesIdx, err := res.NameToColumn("adjClose")
		if err != nil {
			return nil, errors.New("Adjsuted close metric not found")
		}
		valueSeries = res.Series[valueSeriesIdx]
	default:
		return nil, errors.New("Un-supported metric")
	}

	if err != nil {
		return nil, err
	}

	valueSeries.Rename(symbol)
	df := dataframe.NewDataFrame(timeSeries, valueSeries)

	return df, err
}
