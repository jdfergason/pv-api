package portfolio

import (
	"math"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/stat"
)

// DrawDown draw down period in portfolio
type DrawDown struct {
	Begin       int64   `json:"begin"`
	End         int64   `json:"end"`
	Recovery    int64   `json:"recovery"`
	LossPercent float64 `json:"lossPercent"`
}

type CAGR struct {
	OneYear   float64 `json:"1-yr"`
	ThreeYear float64 `json:"3-yr"`
	FiveYear  float64 `json:"5-yr"`
	TenYear   float64 `json:"10-yr"`
}

// MetricsBundle collection of statistics for a portfolio
type MetricsBundle struct {
	CAGRS         CAGR        `json:"cagrs"`
	DrawDowns     []*DrawDown `json:"drawDowns"`
	SharpeRatio   float64     `json:"sharpeRatio"`
	SortinoRatio  float64     `json:"sortinoRatio"`
	StdDev        float64     `json:"stdDev"`
	UlcerIndexAvg float64     `json:"ulcerIndexAvg"`
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// BuildMetricsBundle calculate standard package of metrics
func (perf *Performance) BuildMetricsBundle() {
	cagrs := CAGR{
		OneYear:   perf.PeriodCagr(1),
		ThreeYear: perf.PeriodCagr(3),
		FiveYear:  perf.PeriodCagr(5),
		TenYear:   perf.PeriodCagr(10),
	}

	bundle := MetricsBundle{
		CAGRS:         cagrs,
		DrawDowns:     perf.DrawDowns(),
		SharpeRatio:   perf.SharpeRatio(),
		SortinoRatio:  perf.SortinoRatio(),
		StdDev:        perf.StdDev(),
		UlcerIndexAvg: perf.AvgUlcerIndex(14),
	}

	perf.MetricsBundle = bundle
}

// DrawDowns compute top 10 draw downs
func (perf *Performance) DrawDowns() []*DrawDown {
	if len(perf.Measurements) <= 0 {
		return []*DrawDown{}
	}

	allDrawDowns := []*DrawDown{}

	var peak float64 = perf.Measurements[0].Value
	var drawDown *DrawDown
	for _, v := range perf.Measurements {
		peak = math.Max(peak, v.Value)
		diff := v.Value - peak
		if diff < 0 {
			if drawDown == nil {
				drawDown = &DrawDown{
					Begin:       v.Time,
					End:         v.Time,
					LossPercent: (v.Value / peak) - 1.0,
				}
			}

			loss := (v.Value/peak - 1.0)
			if loss < drawDown.LossPercent {
				drawDown.End = v.Time
				drawDown.LossPercent = loss
			}
		} else if drawDown != nil {
			drawDown.Recovery = v.Time
			allDrawDowns = append(allDrawDowns, drawDown)
			drawDown = nil
		}
	}

	sort.Slice(allDrawDowns, func(i, j int) bool {
		return allDrawDowns[i].LossPercent < allDrawDowns[j].LossPercent
	})

	return allDrawDowns[0:min(10, len(allDrawDowns))]
}

// OneDayReturn compute the return over the last day
func (perf *Performance) OneDayReturn(forDate time.Time, p *Portfolio) float64 {
	// Compute 1-day return
	value := perf.Measurements
	sz := len(value)
	var todaysValue float64
	if sz > 0 {
		todaysValue = value[sz-1].Value
	}

	yesterdayValue, err := p.ValueAsOf(forDate.AddDate(0, 0, -1))
	if err != nil {
		log.WithFields(log.Fields{
			"TargetDate": forDate.AddDate(0, 0, -1),
			"Function":   "cmd/notifier/main.go:oneDayReturn",
			"Error":      err,
		}).Error("Cannot get value of portfolio for date")
	}

	if yesterdayValue > 0 {
		return todaysValue/yesterdayValue - 1.0
	}

	return 0
}

// OneWeekReturn compute the return over one week
func (perf *Performance) OneWeekReturn(forDate time.Time, p *Portfolio) float64 {
	// Compute 1-day return
	value := perf.Measurements
	sz := len(value)
	var todaysValue float64
	if sz > 0 {
		todaysValue = value[sz-1].Value
	}

	lastWeekValue, err := p.ValueAsOf(forDate.AddDate(0, 0, -7))
	if err != nil {
		log.WithFields(log.Fields{
			"TargetDate": forDate.AddDate(0, 0, -7),
			"Function":   "cmd/notifier/main.go:oneDayReturn",
			"Error":      err,
		}).Error("Cannot get value of portfolio for date")
	}

	if lastWeekValue > 0 {
		return todaysValue/lastWeekValue - 1.0
	}

	return 0
}

// OneMonthReturn get one month return from performance measurement
func (perf *Performance) OneMonthReturn(forDate time.Time) float64 {
	value := perf.Measurements
	sz := len(value)
	for ii := sz - 1; ii >= 0; ii-- {
		dt := time.Unix(value[ii].Time, 0)
		year, month, day := dt.Date()
		dt = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		if forDate.Equal(dt) {
			return value[ii].PercentReturn
		}
	}

	log.WithFields(log.Fields{
		"Function": "cmd/notifier/main.go:oneMonthReturn",
	}).Error("Could not find one-month return for requested date")
	return 0
}

// NetProfit total profit earned on portfolio
func (perf *Performance) NetProfit() float64 {
	return perf.Measurements[len(perf.Measurements)-1].Value - perf.TotalDeposited + perf.TotalWithdrawn
}

// NetProfitPercent profit earned on portfolio expressed as a percent
func (perf *Performance) NetProfitPercent() float64 {
	return perf.NetProfit() / perf.TotalDeposited
}

// PeriodCagr (final/initial)^(1/period) - 1
// period is number of years to calculate CAGR over
func (perf *Performance) PeriodCagr(period int) float64 {
	var final float64 = perf.Measurements[len(perf.Measurements)-1].Value
	var initial float64

	finalDate := time.Unix(perf.Measurements[len(perf.Measurements)-1].Time, 0)
	initialDate := finalDate.AddDate(-1*period, 0, 0)
	for ii := len(perf.Measurements) - 1; ii >= 0; ii-- {
		meas := perf.Measurements[ii]
		t := time.Unix(meas.Time, 0)
		if t.Before(initialDate) || t.Equal(initialDate) {
			initial = meas.Value
			break
		}
	}

	if initial == 0 {
		return 0.0
	}

	return math.Pow(final/initial, 1.0/float64(period)) - 1.0
}

// Std standard deviation of portfolio
func (perf *Performance) StdDev() float64 {
	N := len(perf.Measurements)
	rets := make([]float64, N)
	for ii, xx := range perf.Measurements {
		rets[ii] = xx.PercentReturn
	}
	m := stat.Mean(rets, nil)

	var stderr float64
	for _, xx := range perf.Measurements {
		stderr += math.Pow(xx.PercentReturn-m, 2)
	}

	return math.Sqrt(stderr/float64(N)) * math.Sqrt(12)
}

// UlcerIndex The Ulcer Index (UI) is a technical indicator that measures downside
// risk in terms of both the depth and duration of price declines. The index
// increases in value as the price moves farther away from a recent high and falls as
// the price rises to new highs. The indicator is usually calculated over a 14-day
// period, with the Ulcer Index showing the percentage drawdown a trader can expect
// from the high over that period.
//
// The greater the value of the Ulcer Index, the longer it takes for a stock to get
// back to the former high. Simply stated, it is designed as one measure of
// volatility only on the downside.
//
// Percentage Drawdown = [(Close - 14-period High Close)/14-period High Close] x 100
// Squared Average = (14-period Sum of Percentage Drawdown Squared)/14
// Ulcer Index = Square Root of Squared Average
//
// period is number of days to lookback
func (perf *Performance) UlcerIndex(period int) []float64 {
	N := len(perf.Measurements)

	if N < period {
		return []float64{0}
	}

	res := make([]float64, N-period)
	lookback := make([]float64, period)
	lookbackIdx := 0

	for ii, xx := range perf.Measurements {
		lookback[lookbackIdx] = xx.Value
		lookbackIdx = (lookbackIdx + 1) % period
		if ii < period {
			continue
		}

		// Find max close over period
		var maxClose float64
		for _, yy := range lookback {
			maxClose = math.Max(maxClose, yy)
		}

		percDD := make([]float64, period)
		var sqSum float64
		for jj, yy := range lookback {
			percDD[jj] = ((yy - maxClose) / maxClose) * 100
			sqSum += math.Pow(percDD[jj], 2)
		}
		sqAvg := sqSum / float64(period)
		res[ii-period] = math.Sqrt(sqAvg)
	}
	return res
}

// AvgUlcerIndex compute average ulcer index
// period is number of days to lookback
func (perf *Performance) AvgUlcerIndex(period int) float64 {
	u := perf.UlcerIndex(period)
	return stat.Mean(u, nil)
}

// ExcessReturn compute the rate of return that is in excess of the risk free rate
func (perf *Performance) ExcessReturn() []float64 {
	rets := make([]float64, len(perf.Measurements))
	prev := perf.Measurements[0].RiskFreeValue
	for ii, xx := range perf.Measurements {
		riskFreeRate := xx.RiskFreeValue/prev - 1.0
		prev = xx.RiskFreeValue
		rets[ii] = xx.PercentReturn - riskFreeRate
	}
	return rets
}

// SharpeRatio The ratio is the average return earned in excess of the risk-free
// rate per unit of volatility or total risk. Volatility is a measure of the price
// fluctuations of an asset or portfolio.
// Sharpe = (Rp - Rf) / std
func (perf *Performance) SharpeRatio() float64 {
	excessReturn := perf.ExcessReturn()
	sharpe := stat.Mean(excessReturn, nil) / stat.StdDev(excessReturn, nil)
	return sharpe * math.Sqrt(12) // annualize rate
}

// SortinoRatio a variation of the Sharpe ratio that differentiates harmful
// volatility from total overall volatility by using the asset's standard deviation
// of negative portfolio returns—downside deviation—instead of the total standard
// deviation of portfolio returns. The Sortino ratio takes an asset or portfolio's
// return and subtracts the risk-free rate, and then divides that amount by the
// asset's downside deviation.
//
// Calculation is based on this paper by Red Rock Capital
// http://www.redrockcapital.com/Sortino__A__Sharper__Ratio_Red_Rock_Capital.pdf
func (perf *Performance) SortinoRatio() float64 {
	// get downside returns
	var downside float64
	excessReturn := perf.ExcessReturn()
	for _, xx := range excessReturn {
		if xx < 0 {
			downside += math.Pow(xx, 2)
		}
	}
	downside = downside / float64(len(excessReturn))
	if downside == 0 {
		return 0
	}
	sortino := stat.Mean(excessReturn, nil) / math.Sqrt(downside)
	return sortino * math.Sqrt(12) // annualize rate by adjusting by month
}

// KRatio The K-ratio is a valuation metric that examines the consistency of an equity's return over time.
// k-ratio = (Slope logVAMI regression line) / n(Standard Error of the Slope)

// VolatilityMonthly

// VolatilityAnnualized

// DownsideDeviation

// USMarketCorrelation

// Beta is a measure of the volatility—or systematic risk—of a security or portfolio
// compared to the market as a whole. Beta is used in the capital asset pricing model
// (CAPM), which describes the relationship between systematic risk and expected
// return for assets (usually stocks). CAPM is widely used as a method for pricing
// risky securities and for generating estimates of the expected returns of assets,
// considering both the risk of those assets and the cost of capital.
func (perf *Performance) Beta(benchmark *Performance) float64 {
	retA := perf.Return()
	retB := benchmark.Return()

	covar := stat.Covariance(retA, retB, nil)
	return covar / stat.Variance(retB, nil)
}

// Alpha

// RSquared

// TreynorRatio also known as the reward-to-volatility ratio, is a performance
// metric for determining how much excess return was generated for each unit of risk
// taken on by a portfolio.
// treynor = Excess Return / Beta
func (perf *Performance) TreynorRatio(benchmark *Performance) float64 {
	excessReturn := perf.ExcessReturn()
	return stat.Mean(excessReturn, nil) / perf.Beta(benchmark)
}

// CalmarRatio

// Return
func (perf *Performance) Return() []float64 {
	rets := make([]float64, len(perf.Measurements))
	for ii, xx := range perf.Measurements {
		rets[ii] = xx.Value
	}
	return rets
}

// TrackingError

// InformationRatio

// Skewness

// ExcessKurtosis

// ValueAtRisk

// UpsideCaptureRatio

// DownsideCaptureRatio

// SafeWithdrawalRate

// PerpetualWithdrawalRate

// NPositivePeriods

// GainLossRatio
