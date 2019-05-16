package internal

import (
    "context"
    "errors"
    "github.com/globalsign/mgo/bson"
    "github.com/golang/protobuf/ptypes"
    "github.com/paysuper/paysuper-currencies-rates/pkg/proto/currencyrates"
    "go.uber.org/zap"
    "strings"
    "time"
)

func (s *Service) GetOxrRate(
    ctx context.Context,
    req *currencyrates.GetRateRequest,
    res *currencyrates.RateData,
) error {
    err := s.getRate(collectionSuffixOxr, req.From, req.To, bson.M{}, res)
    if err != nil {
        zap.S().Errorw(errorCurrentRateRequest, "error", err, "req", req)
        return err
    }
    s.applyCorrection(res, collectionSuffixOxr, req.MerchantId)
    return nil
}

func (s *Service) GetCentralBankRateForDate(
    ctx context.Context,
    req *currencyrates.GetCentralBankRateRequest,
    res *currencyrates.RateData,
) error {

    dt, err := ptypes.Timestamp(req.Datetime)
    if err != nil {
        zap.S().Errorw(errorDatetimeConversion, "error", err, "req", req)
        return err
    }

    query := bson.M{"created_at": bson.M{"$lte": s.Eod(dt)}}
    err = s.getRate(collectionSuffixCb, req.From, req.To, query, res)
    if err != nil {
        zap.S().Errorw(errorCentralBankRateRequest, "error", err, "req", req)
        return err
    }

    s.applyCorrection(res, collectionSuffixCb, req.MerchantId)
    return nil
}

func (s *Service) GetPaysuperRate(
    ctx context.Context,
    req *currencyrates.GetRateRequest,
    res *currencyrates.RateData,
) error {
    err := s.getRate(collectionSuffixPaysuper, req.From, req.To, bson.M{}, res)
    if err != nil {
        zap.S().Errorw(errorCurrentRateRequest, "error", err, "req", req)
        return err
    }
    s.applyCorrection(res, collectionSuffixPaysuper, req.MerchantId)
    return nil
}

func (s *Service) GetStockRate(
    ctx context.Context,
    req *currencyrates.GetRateRequest,
    res *currencyrates.RateData,
) error {
    err := s.getRate(collectionSuffixStock, req.From, req.To, bson.M{}, res)
    if err != nil {
        zap.S().Errorw(errorCurrentRateRequest, "error", err, "req", req)
        return err
    }
    s.applyCorrection(res, collectionSuffixStock, req.MerchantId)
    return nil
}

func (s *Service) GetCardpayRate(
    ctx context.Context,
    req *currencyrates.GetRateRequest,
    res *currencyrates.RateData,
) error {
    err := s.getRate(collectionSuffixCardpay, req.From, req.To, bson.M{}, res)
    if err != nil {
        zap.S().Errorw(errorCurrentRateRequest, "error", err, "req", req)
        return err
    }
    s.applyCorrection(res, collectionSuffixCardpay, req.MerchantId)
    return nil
}

func (s *Service) SetPaysuperCorrectionCorridor(
    ctx context.Context,
    req *currencyrates.CorrectionCorridor,
    res *currencyrates.EmptyResponse,
) error {

    corridor := PaysuperCorridor{
        Value:     req.Value,
        CreatedAt: time.Now(),
    }

    err := s.db.Collection(collectionNamePaysuperCorridors).Insert(corridor)
    if err != nil {
        zap.S().Errorw(errorDbInsertFailed, "error", err, "data", corridor)
        return err
    }

    return nil
}

func (s *Service) AddRateCorrectionRule(
    ctx context.Context,
    req *currencyrates.CorrectionRule,
    res *currencyrates.EmptyResponse,
) error {

    req.Id = bson.NewObjectId().Hex()
    req.CreatedAt = ptypes.TimestampNow()
    req.RateType = strings.ToLower(req.RateType)

    if err := s.validateReq(req); err != nil {
        zap.S().Errorw(errorDbReqInvalid, "error", err, "req", req)
        return err
    }

    if len(req.PairCorrection) > 0 {
        for pair := range req.PairCorrection {
            if !s.isPairExists(pair) {
                zap.S().Errorw(errorCurrencyPairNotExists, "req", req, "pair", pair)
                return errors.New(errorCurrencyPairNotExists)
            }
        }
    }

    err := s.db.Collection(collectionNameCorrectionRules).Insert(req)
    if err != nil {
        zap.S().Errorw(errorDbInsertFailed, "error", err, "req", req)
        return err
    }

    return nil
}

func (s *Service) GetRateCorrectionRule(
    ctx context.Context,
    req *currencyrates.CorrectionRuleRequest,
    res *currencyrates.CorrectionRule,
) error {
    if err := s.validateReq(req); err != nil {
        zap.S().Errorw(errorCorrectionRuleRequestInvalid, "error", err, "req", req)
        return err
    }

    err := s.getCorrectionRule(req.RateType, req.MerchantId, res)
    if err != nil {
        zap.S().Errorw(errorCorrectionRuleNotFound, "error", err, "req", req)
        return err
    }
    return nil
}
