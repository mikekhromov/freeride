package approve

import (
	"context"
	"fmt"

	"freeride/vpn-bot/config"
	"freeride/vpn-bot/services/hiddify"
	"freeride/vpn-bot/store"
)

type Service struct {
	Store   *store.Store
	Hiddify *hiddify.Client
	Cfg     config.Config
}

func (s *Service) ApproveUser(ctx context.Context, userID int64, adminTelegramID int64) (tgUser string, tgID int64, hLink, mtLink string, alreadyActive bool, err error) {
	u, err := s.Store.GetUserByID(ctx, userID)
	if err != nil {
		return "", 0, "", "", false, err
	}
	if u.Status == "banned" {
		return "", 0, "", "", false, fmt.Errorf("пользователь заблокирован")
	}
	if u.Status == "active" {
		if u.HiddifyUUID != "" {
			h, e1 := s.Hiddify.ProfileURLByUUID(ctx, u.HiddifyUUID)
			mt, e2 := s.Hiddify.MTProxyLinkByUUID(ctx, u.HiddifyUUID)
			if e1 == nil && e2 == nil {
				return u.TelegramUsername, u.TelegramID, h, mt, true, nil
			}
		}
	}
	if u.Status != "pending" && u.Status != "active" {
		return "", 0, "", "", false, fmt.Errorf("статус пользователя не позволяет одобрить: %s", u.Status)
	}

	name := fmt.Sprintf("tg-%d", u.TelegramID)
	if u.TelegramUsername != "" {
		name = u.TelegramUsername
	}
	hUID, err := s.Hiddify.CreateUser(ctx, name, s.Cfg.UserPackageDays, s.Cfg.UserUsageLimitGB)
	if err != nil {
		return "", 0, "", "", false, err
	}
	hSub, err := s.Hiddify.ProfileURLByUUID(ctx, hUID)
	if err != nil {
		return "", 0, "", "", false, err
	}
	mtLink, err = s.Hiddify.MTProxyLinkByUUID(ctx, hUID)
	if err != nil {
		return "", 0, "", "", false, err
	}

	if err := s.Store.ActivateUser(ctx, userID, adminTelegramID, hUID); err != nil {
		return "", 0, "", "", false, err
	}

	return u.TelegramUsername, u.TelegramID, hSub, mtLink, false, nil
}

func (s *Service) ReissueForTelegramUser(ctx context.Context, telegramID int64) (tgUser string, tgID int64, hLink, mtLink string, err error) {
	u, err := s.Store.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return "", 0, "", "", err
	}
	if u.Status != "active" {
		return "", 0, "", "", fmt.Errorf("перевыпуск доступен только для active")
	}
	if u.HiddifyUUID != "" {
		if err := s.Hiddify.DeleteUser(ctx, u.HiddifyUUID); err != nil {
			return "", 0, "", "", err
		}
	}
	name := fmt.Sprintf("tg-%d", u.TelegramID)
	if u.TelegramUsername != "" {
		name = u.TelegramUsername
	}
	hUID, err := s.Hiddify.CreateUser(ctx, name, s.Cfg.UserPackageDays, s.Cfg.UserUsageLimitGB)
	if err != nil {
		return "", 0, "", "", err
	}
	hSub, err := s.Hiddify.ProfileURLByUUID(ctx, hUID)
	if err != nil {
		return "", 0, "", "", err
	}
	mt, err := s.Hiddify.MTProxyLinkByUUID(ctx, hUID)
	if err != nil {
		return "", 0, "", "", err
	}
	if err := s.Store.ActivateUser(ctx, u.ID, telegramID, hUID); err != nil {
		return "", 0, "", "", err
	}
	return u.TelegramUsername, u.TelegramID, hSub, mt, nil
}
