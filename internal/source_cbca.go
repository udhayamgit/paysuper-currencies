package internal

import (
    "errors"
    "fmt"
    currencyrates "github.com/paysuper/paysuper-currencies-rates/proto"
    "go.uber.org/zap"
    "net/http"
    "time"
)

const (
    errorCbcaUrlValidationFailed   = "CBCA Rates url validation failed"
    errorCbcaRequestFailed         = "CBCA Rates request failed"
    errorCbcaResponseParsingFailed = "CBCA Rates response parsing failed"
    errorCbcaSaveRatesFailed       = "CBCA Rates save data failed"
    errorCbcaNoResults             = "CBCA Rates no results"
    errorRateDataNotFound          = "CBCA Rate data not found"
    errorRateDataInvalidFormat     = "CBCA Rate data has invalid format"

    cbcaTo          = "CAD"
    cbcaSource      = "CBCA"
    cbcaUrlTemplate = "https://www.bankofcanada.ca/valet/observations/group/FX_RATES_DAILY/json?start_date=%s"

    cbcaKeyMask = "FX%s%s"
)

type CbcaResponse struct {
    Observations []map[string]interface{} `json:"observations"`
}

func (s *Service) RequestRatesCbca() error {
    zap.S().Info("Requesting rates from CBCA")

    headers := map[string]string{
        HeaderContentType: MIMEApplicationJSON,
        HeaderAccept:      MIMEApplicationJSON,
    }

    today := time.Now()
    d := today.AddDate(0, 0, -7)

    reqUrl, err := s.validateUrl(fmt.Sprintf(cbcaUrlTemplate, d.Format(dateFormatLayout)))
    if err != nil {
        zap.S().Errorw(errorCbcaUrlValidationFailed, "error", err)
        s.sendCentrifugoMessage(errorCbcaUrlValidationFailed, err)
        return err
    }

    println("blah reqUrl.String()", reqUrl.String())

    resp, err := s.request(http.MethodGet, reqUrl.String(), nil, headers)

    if err != nil {
        zap.S().Errorw(errorCbcaRequestFailed, "error", err)
        s.sendCentrifugoMessage(errorCbcaRequestFailed, err)
        return err
    }

    res := &CbcaResponse{}
    err = s.decodeJson(resp, res)

    if err != nil {
        zap.S().Errorw(errorCbcaResponseParsingFailed, "error", err)
        s.sendCentrifugoMessage(errorCbcaResponseParsingFailed, err)
        return err
    }

    err = s.processRatesCbca(res)

    if err != nil {
        zap.S().Errorw(errorCbcaSaveRatesFailed, "error", err)
        s.sendCentrifugoMessage(errorCbcaSaveRatesFailed, err)
        return err
    }

    zap.S().Info("Rates from CBCA updated")

    return nil
}

func (s *Service) processRatesCbca(res *CbcaResponse) error {

    if len(res.Observations) == 0 {
        return errors.New(errorCbcaNoResults)
    }

    rates := []*currencyrates.RateData{}

    lastRates := res.Observations[len(res.Observations)-1]

    for _, cFrom := range s.cfg.CbrfBaseCurrencies {
        key := fmt.Sprintf(cbcaKeyMask, cFrom, cbcaTo)

        rateItem, ok := lastRates[key]
        if !ok {
            return errors.New(errorRateDataNotFound)
        }

        rawRate, ok := rateItem.(map[string]interface{})["v"]
        if !ok {
            return errors.New(errorRateDataInvalidFormat)
        }

        rate := rawRate.(float64)

        // direct pair
        rates = append(rates, &currencyrates.RateData{
            Pair:   cFrom + cbcaTo,
            Rate:   rate,
            Source: cbcaSource,
        })

        // inverse pair
        rates = append(rates, &currencyrates.RateData{
            Pair:   cbcaTo + cFrom,
            Rate:   1 / rate,
            Source: cbcaSource,
        })
    }

    err := s.saveRates(collectionSuffixCb, rates)
    if err != nil {
        return err
    }
    return nil
}
