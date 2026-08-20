package http

import "time"

const sessionCookie = "ehd_session"

// --- запросы ---

type registerReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	IIN      string `json:"iin"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type changePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type edsVerifyReq struct {
	Challenge  string `json:"challenge"`
	SignedData string `json:"signed_data"`
}

type createRoleReq struct {
	Code   string `json:"code"`
	NameRu string `json:"name_ru"`
	NameKk string `json:"name_kk"`
}

type updateUserReq struct {
	IINVerified   *bool     `json:"iin_verified"`
	Status        *string   `json:"status"`
	RoleIDs       *[]string `json:"role_ids"`
	RegionIDs     *[]string `json:"region_ids"`
	DepartmentIDs *[]string `json:"department_ids"`
}

// --- ответы ---

type loginResp struct {
	UserID                 string    `json:"user_id"`
	Login                  string    `json:"login"`
	ExpiresAt              time.Time `json:"expires_at"`
	PasswordChangeRequired bool      `json:"password_change_required"`
}

type meResp struct {
	UserID          string   `json:"user_id"`
	Login           string   `json:"login"`
	IsAdmin         bool     `json:"is_admin"`
	Roles           []string `json:"roles"`
	RegionCodes     []string `json:"region_codes"`
	DepartmentCodes []string `json:"department_codes"`
}

type userView struct {
	ID              string    `json:"id"`
	Login           string    `json:"login"`
	Email           string    `json:"email"`
	FullName        string    `json:"full_name"`
	IINMasked       string    `json:"iin_masked"`
	IINVerified     bool      `json:"iin_verified"`
	Status          string    `json:"status"`
	FailedAttempts  int       `json:"failed_attempts"`
	Roles           []string  `json:"roles"`
	RegionCodes     []string  `json:"region_codes"`
	DepartmentCodes []string  `json:"department_codes"`
	CreatedAt       time.Time `json:"created_at"`
}

type roleView struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	NameRu string `json:"name_ru"`
	NameKk string `json:"name_kk"`
	Status string `json:"status"`
}

type referenceView struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	NameRu string `json:"name_ru"`
	NameKk string `json:"name_kk"`
	Status string `json:"status"`
}

func emptyIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
