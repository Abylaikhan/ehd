package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Виды и статусы шагов маршрута (спека 010 §3).
const (
	RouteKindApprove  = "approve"  // согласование (последний согласующий = подписант, В1)
	RouteKindOutgoing = "outgoing" // создать исходящее

	RouteStepDraft = "draft" // шаблон маршрута (round=0)
	RouteStepOpen  = "open"  // задача у исполнителя
	RouteStepDone  = "done"

	RouteResultApproved = "approved"
	RouteResultRejected = "rejected"
)

// Ошибки согласования (спека 010).
var (
	// ErrRouteIncomplete — маршрут не заполнен: нет согласующего или ответственного (AT-09).
	ErrRouteIncomplete = errors.New("маршрут не заполнен")
	// ErrRouteStateInvalid — действие недопустимо в текущем статусе уведомления.
	ErrRouteStateInvalid = errors.New("действие недоступно в текущем статусе уведомления")
	// ErrNotAssignee — пользователь не назначен исполнителем открытого шага (403).
	ErrNotAssignee = errors.New("вы не являетесь исполнителем текущего этапа")
	// ErrCommentRequired — возврат на доработку требует комментария (FR-6).
	ErrCommentRequired = errors.New("для возврата на доработку обязателен комментарий")
	// ErrNumberingUnavailable — у департамента не настроен код региона (FR-8).
	ErrNumberingUnavailable = errors.New("нумерация недоступна: у департамента не задан код региона")

	// Подписание согласующим (спека 015).
	// ErrSignatureRequired — режим EVGA_APPROVE_EDS включён, но подпись не передана (FR-2).
	ErrSignatureRequired = errors.New("для подписания требуется ЭЦП")
	// ErrChallengeInvalid — challenge не выдавался, истёк или уже использован (FR-1).
	ErrChallengeInvalid = errors.New("запрос на подпись недействителен или истёк")
	// ErrSignatureInvalid — CMS не прошла криптопроверку (FR-2).
	ErrSignatureInvalid = errors.New("подпись не прошла проверку")
	// ErrSignerMismatch — ИИН сертификата не совпадает с ИИН пользователя (FR-2).
	ErrSignerMismatch = errors.New("сертификат принадлежит другому лицу")
)

// RouteStep — этап маршрута (хранится в БД ЕХД).
type RouteStep struct {
	ID            int64
	NoticeID      int64
	Round         int
	StepNN        int
	Kind          string
	AssigneeObmID int64
	AssigneeName  string
	Status        string
	Result        string
	Comment       string
	OpenedAt      *time.Time
	ClosedAt      *time.Time
	// Подпись согласующего (спека 015 FR-3/FR-4): сам CMS наружу не отдаётся.
	SignedAt *time.Time
}

// RouteInput — редактирование шаблона маршрута (FR-2).
type RouteInput struct {
	ApproverIDs    []int64
	OutgoingUserID int64
}

// Validate — маршрут заполнен: ≥1 согласующий и ответственный за исходящее (AT-09).
func (r RouteInput) Validate() error {
	if len(r.ApproverIDs) == 0 {
		return fmt.Errorf("%w: не указан согласующий", ErrRouteIncomplete)
	}
	seen := map[int64]bool{}
	for _, id := range r.ApproverIDs {
		if id <= 0 {
			return fmt.Errorf("%w: некорректный согласующий", ErrRouteIncomplete)
		}
		if seen[id] {
			return fmt.Errorf("%w: согласующий указан дважды", ErrRouteIncomplete)
		}
		seen[id] = true
	}
	if r.OutgoingUserID <= 0 {
		return fmt.Errorf("%w: не назначен ответственный за создание исходящего", ErrRouteIncomplete)
	}
	return nil
}

// FormatNoticeNumber — номер уведомления `<region_code>/<год>/<NNNNNN>` (EVGA-FR-070).
func FormatNoticeNumber(regionCode string, year int, seq int64) (string, error) {
	regionCode = strings.TrimSpace(regionCode)
	if regionCode == "" {
		return "", ErrNumberingUnavailable
	}
	if seq <= 0 || seq > 999999 {
		return "", fmt.Errorf("порядковый номер вне диапазона: %d", seq)
	}
	return fmt.Sprintf("%s/%d/%06d", regionCode, year, seq), nil
}

// YearFromNoticeNumber — год из номера уведомления «<регион>/<год>/<NNNNNN>».
// Источник года для годового сброса: колонки its_departments.notice_num_year
// в боевой БД нет (спека 021), а номер есть всегда и лежит в той же транзакции.
// Нераспознанный номер → nil (трактуется как «другой год» → счёт с 1).
func YearFromNoticeNumber(docnum string) *int32 {
	parts := strings.Split(strings.TrimSpace(docnum), "/")
	if len(parts) != 3 {
		return nil
	}
	y, err := strconv.Atoi(parts[1])
	if err != nil || y < 2000 || y > 2100 {
		return nil
	}
	v := int32(y)
	return &v
}

// NextNoticeSeq — следующий порядковый номер с годовым сбросом (EVGA-FR-072/073):
// counterYear ≠ текущий год → счёт с 1; иначе counter+1.
func NextNoticeSeq(counter int64, counterYear *int32, nowYear int) int64 {
	if counterYear == nil || int(*counterYear) != nowYear {
		return 1
	}
	return counter + 1
}

// ObmUser — сотрудник департамента из obm_evga.users (участник маршрута, FR-3).
type ObmUser struct {
	ID    int64
	Name  string
	Login string
}
