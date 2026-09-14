package esedo

import (
	"context"
	"errors"
)

// Sender — отправка исходящего в ЕСЭДО. Реализации: StubSender (безопасная заглушка, по
// умолчанию) и SoapSender (скелет клиента ШЭП/ВШЭП с отключённым транспортом).
type Sender interface {
	// UploadAttachment загружает файл в ХЭД (MIND-S-0048) и возвращает fileIdentifier.
	UploadAttachment(ctx context.Context, a Attachment) (fileID string, err error)
	// SendOutgoing отправляет docOutgoing через MIND-S-2147 SendMessage и возвращает квитанцию.
	SendOutgoing(ctx context.Context, doc DocOutgoing) (SendResult, error)
}

// Signer — формирование CMS-подписи ГОСТ (сертификат юрлица) для secondSignData (OQ-05).
// Реальная реализация — через NCANode-сайдкар (ГОСТ РК) после получения сертификата.
type Signer interface {
	SignCMS(ctx context.Context, data []byte) (cms []byte, err error)
}

// Config — реквизиты подключения к ШЭП/ВШЭП. Заполняются после выдачи учётки (Бауыржан).
// Enabled — ЖЁСТКИЙ рубильник; по умолчанию false: даже собранный запрос не отправляется.
type Config struct {
	Enabled   bool   // включение реального клиента (по умолчанию false)
	Endpoint  string // адрес SendMessage через ШЭП/ВШЭП
	SenderID  string // requestInfo.sender.senderId
	Password  string // requestInfo.sender.password
	ServiceID string // requestInfo.serviceId (по умолчанию ESEDO_UNIVERSAL_SERVICE)
	RouteID   string // requestInfo.routeId (по умолчанию R_ESEDO)
	FromOrg   string // код организации-отправителя (metadataSystem.from/senderOrg)
	CertPath  string // серверный сертификат ГОСТ (WSSE-подпись конверта госслужащим)
}

// DefaultServiceID — идентификатор универсального сервиса ЕСЭДО (MIND-S-2147).
const DefaultServiceID = "ESEDO_UNIVERSAL_SERVICE"

// DefaultRouteID — маршрут ЕСЭДО из шаблона аналитика (routeId=R_ESEDO).
const DefaultRouteID = "R_ESEDO"

var (
	// ErrDisabled — отправка выключена (Config.Enabled=false): нет доступов/согласования.
	ErrDisabled = errors.New("esedo: отправка в ЕСЭДО отключена (Config.Enabled=false)")
	// ErrNotConfigured — включено, но не заданы обязательные реквизиты (endpoint/senderId).
	ErrNotConfigured = errors.New("esedo: не заданы реквизиты подключения к ШЭП")
	// ErrTransportDisabled — реальный сетевой вызов намеренно отключён в заготовке
	// (ограничение «в ЕСЭДО запросы не отправлять»): конверт собран, но не отправлен.
	ErrTransportDisabled = errors.New("esedo: реальный транспорт намеренно отключён (заготовка spec 014)")
)
