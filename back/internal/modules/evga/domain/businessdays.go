package domain

import "time"

// AddBusinessDays — дата через n будних дней от from (суббота/воскресенье пропускаются).
// Праздники РК не учитываются — решение аналитика №3-5 от 08.09.2026:
// «календаря не будет, автоматом ставим 10 рабочих дней».
func AddBusinessDays(from time.Time, n int) time.Time {
	d := from
	for added := 0; added < n; {
		d = d.AddDate(0, 0, 1)
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			added++
		}
	}
	return d
}
