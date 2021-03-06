package strategies_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"main/data"
	"main/strategies"
	"time"

	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Daa", func() {
	var (
		daa     *strategies.KellersDefensiveAssetAllocation
		manager data.Manager
	)

	BeforeEach(func() {
		jsonParams := `{"riskUniverse": ["VFINX", "PRIDX"], "cashUniverse": ["VUSTX"], "protectiveUniverse": ["VUSTX"], "breadth": 1, "topT": 1}`
		params := map[string]json.RawMessage{}
		if err := json.Unmarshal([]byte(jsonParams), &params); err != nil {
			panic(err)
		}

		tmp, err := strategies.NewKellersDefensiveAssetAllocation(params)
		if err != nil {
			panic(err)
		}
		daa = tmp.(*strategies.KellersDefensiveAssetAllocation)

		manager = data.NewManager(map[string]string{
			"tiingo": "TEST",
		})

		content, err := ioutil.ReadFile("testdata/TB3MS.csv")
		if err != nil {
			panic(err)
		}

		httpmock.RegisterResponder("GET", "https://fred.stlouisfed.org/graph/fredgraph.csv?mode=fred&id=TB3MS&cosd=1979-07-01&coed=2021-01-01&fq=AdjustedClose&fam=avg",
			httpmock.NewBytesResponder(200, content))

		content, err = ioutil.ReadFile("testdata/VUSTX.csv")
		if err != nil {
			panic(err)
		}

		httpmock.RegisterResponder("GET", "https://api.tiingo.com/tiingo/daily/VUSTX/prices?startDate=1979-01-01&endDate=2021-01-01&format=csv&resampleFreq=Monthly&token=TEST",
			httpmock.NewBytesResponder(200, content))

		content, err = ioutil.ReadFile("testdata/VUSTX_2.csv")
		if err != nil {
			panic(err)
		}

		httpmock.RegisterResponder("GET", "https://api.tiingo.com/tiingo/daily/VUSTX/prices?startDate=1990-01-31&endDate=2021-01-01&format=csv&resampleFreq=Monthly&token=TEST",
			httpmock.NewBytesResponder(200, content))

		content, err = ioutil.ReadFile("testdata/VFINX.csv")
		if err != nil {
			panic(err)
		}

		httpmock.RegisterResponder("GET", "https://api.tiingo.com/tiingo/daily/VFINX/prices?startDate=1979-01-01&endDate=2021-01-01&format=csv&resampleFreq=Monthly&token=TEST",
			httpmock.NewBytesResponder(200, content))

		content, err = ioutil.ReadFile("testdata/VFINX_2.csv")
		if err != nil {
			panic(err)
		}

		httpmock.RegisterResponder("GET", "https://api.tiingo.com/tiingo/daily/VFINX/prices?startDate=1990-01-31&endDate=2021-01-01&format=csv&resampleFreq=Monthly&token=TEST",
			httpmock.NewBytesResponder(200, content))

		content, err = ioutil.ReadFile("testdata/PRIDX.csv")
		if err != nil {
			panic(err)
		}

		httpmock.RegisterResponder("GET", "https://api.tiingo.com/tiingo/daily/PRIDX/prices?startDate=1979-01-01&endDate=2021-01-01&format=csv&resampleFreq=Monthly&token=TEST",
			httpmock.NewBytesResponder(200, content))

		content, err = ioutil.ReadFile("testdata/PRIDX_2.csv")
		if err != nil {
			panic(err)
		}

		httpmock.RegisterResponder("GET", "https://api.tiingo.com/tiingo/daily/PRIDX/prices?startDate=1990-01-31&endDate=2021-01-01&format=csv&resampleFreq=Monthly&token=TEST",
			httpmock.NewBytesResponder(200, content))

		content, err = ioutil.ReadFile("testdata/riskfree.csv")
		if err != nil {
			panic(err)
		}

		today := time.Now()
		url := fmt.Sprintf("https://fred.stlouisfed.org/graph/fredgraph.csv?mode=fred&id=DTB3&cosd=1970-01-01&coed=%d-%02d-%02d&fq=Daily&fam=avg", today.Year(), today.Month(), today.Day())
		httpmock.RegisterResponder("GET", url,
			httpmock.NewBytesResponder(200, content))

		data.InitializeDataManager()
	})

	Describe("Compute momentum scores", func() {
		Context("with full stock history", func() {
			It("should be invested in VUSTX", func() {
				manager.Begin = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
				manager.End = time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
				p, err := daa.Compute(&manager)
				Expect(err).To(BeNil())

				Expect(p.Transactions).Should(HaveLen(701))

				perf, err := p.CalculatePerformance(manager.End)
				Expect(err).To(BeNil())
				Expect(daa.CurrentSymbol).To(Equal("VUSTX"))

				var begin int64
				begin = 633744000
				Expect(perf.PeriodStart).To(Equal(begin))

				var end int64
				end = 1609459200
				Expect(perf.PeriodEnd).To(Equal(end))
				Expect(perf.Measurements).Should(HaveLen(379))

				// Note: perf starts earlier than it should just because the test data starts earlier
				// So we adjust here and ignore the first 6 entries
				Expect(perf.Measurements[6].Time).To(BeNumerically("==", 633744000))
				Expect(perf.Measurements[6].Value).To(BeNumerically("==", 10000))
				Expect(perf.Measurements[6].Holdings).To(Equal("VUSTX"))

				Expect(perf.Measurements[10].Time).To(BeNumerically("==", 644112000))
				Expect(perf.Measurements[10].Value).Should(BeNumerically("~", 10092.8205, 1e-4))
				Expect(perf.Measurements[10].Holdings).To(Equal("PRIDX"))
				Expect(perf.Measurements[10].PercentReturn).Should(BeNumerically("~", 0.0451, 1e-4))

				Expect(perf.Measurements[65].Time).To(BeNumerically("==", 788745600))
				Expect(perf.Measurements[65].Value).Should(BeNumerically("~", 14016.5776, 1e-4))
				Expect(perf.Measurements[65].Holdings).To(Equal("VFINX"))
				Expect(perf.Measurements[65].PercentReturn).Should(BeNumerically("~", 0.0159, 1e-4))

				Expect(perf.Measurements[264].Time).To(BeNumerically("==", 1311897600))
				Expect(perf.Measurements[264].Value).Should(BeNumerically("~", 56807.9076, 1e-4))
				Expect(perf.Measurements[264].Holdings).To(Equal("PRIDX"))
				Expect(perf.Measurements[264].PercentReturn).Should(BeNumerically("~", 0.0418, 1e-4))

				Expect(perf.Measurements[378].Time).To(BeNumerically("==", 1611878400))
				Expect(perf.Measurements[378].Value).Should(BeNumerically("~", 208158.8420, 1e-4))
				Expect(perf.Measurements[378].Holdings).To(Equal("VUSTX"))
				Expect(perf.Measurements[378].PercentReturn).Should(BeNumerically("~", -0.0299, 1e-4))
			})
		})
	})
})
