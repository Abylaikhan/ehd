package domain

import "errors"

// Доменные ошибки Auth Module. Транспорт маппит их в коды/HTTP-статусы.
var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserBlocked         = errors.New("user is blocked")
	ErrUserNotActive       = errors.New("user is not active")
	ErrIINTaken            = errors.New("iin already registered")
	ErrLoginTaken          = errors.New("login already taken")
	ErrWeakPassword        = errors.New("weak password")
	ErrInvalidIIN          = errors.New("invalid iin")
	ErrSessionInvalid      = errors.New("session invalid or expired")
	ErrTempPasswordExpired = errors.New("temporary password expired")
	ErrNotFound            = errors.New("not found")
	ErrLastAdmin           = errors.New("cannot block or demote the last active administrator")
	ErrChallengeInvalid    = errors.New("eds challenge invalid or expired")
	ErrEDSVerification     = errors.New("eds signature verification failed")
)
