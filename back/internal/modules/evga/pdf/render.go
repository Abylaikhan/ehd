// Package pdf — генерация PDF уведомления строго по утверждённому образцу
// (tz/obm-evga-5-15a/Уведомление_об_устранении_нарушения.pdf; спека 012).
// Альбомный A4, тёмно-синий текст, водяной знак на каждой странице,
// четыре секции: каз. уведомление, каз. приложение, рус. уведомление, рус. приложение.
package pdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

//go:embed assets/DejaVuSans.ttf
var fontRegular []byte

//go:embed assets/DejaVuSans-Bold.ttf
var fontBold []byte

//go:embed assets/DejaVuSans-Oblique.ttf
var fontOblique []byte

//go:embed assets/emblem.png
var emblemPNG []byte

//go:embed assets/watermark.png
var watermarkPNG []byte

// Утверждённые тексты (из образца; рус. совпадает с ТЗ §9.2).
const (
	textKZ = "Онлайн бюджеттік мониторингтің нәтижесінде, осы хабарламаға қосымшада көрсетілген жағдайлар бойынша бюджет шығыстарының жоғары тәуекелдері анықталды.\n" +
		"Хабарлама кезінде жеке тұлғалардың карт-шоттарына заңсыз аударым белгілері байқалады.\n" +
		"Бюджет қаражатын негізсіз пайдаланудың жолын кесу мақсатында 10 (он) жұмыс күнінен кешіктірілмейтін мерзімде тәуекелдерді өз бетінше жою ұсынылады.\n" +
		"Олай болмаған жағдайда, нұсқамалық ден қою шараларын қабылдау мәселесі қаралатын болады."
	textRU = "Онлайн бюджетным мониторингом идентифицированы высокие риски в расходной части бюджета по случаям, указанным в приложении к настоящему уведомлению.\n" +
		"На момент уведомления наблюдаются признаки неправомерного перечисления на карт-счета физических лиц.\n" +
		"С целью пресечения необоснованного использования бюджетных средств рекомендуется самостоятельное устранение рисков в срок не позднее 10 (десяти) рабочих дней.\n" +
		"В противном случае будет рассмотрен вопрос о принятии директивных мер реагирования."

	headKZ = "Қазақстан Республикасы\nҚаржы министрлігі\nІшкі мемлекеттік аудит\nкомитеті"
	headRU = "Министерство финансов\nРеспублики Казахстан\nКомитет внутреннего\nгосударственного аудита"
)

// Row — строка приложения (из снимка, EVGA-FR-045).
type Row struct {
	GU          string
	PpoPp       string
	PaymentDate *time.Time
	FM          string
	NM          string
	FT          string
	IIN         string
	LA1         string
	AmountPart  string
}

// Data — данные документа (спека 012 FR-3).
type Data struct {
	DocNum         string // «б/н» у проекта
	DocDate        time.Time
	RecipientRU    string
	RecipientKZ    string
	RecipientCode  string
	DeptTitle      string // «ДВГА по … области КВГА МФ РК»
	SignerPosition string
	SignerPosKZ    string
	SignerName     string // пусто у проекта → прочерк
	ExecName       string
	ExecPhone      string
	ExecEmail      string
	Rows           []Row
}

// цвет текста образца (тёмно-синий)
const inkR, inkG, inkB = 23, 54, 93

type renderer struct{ p *fpdf.Fpdf }

// Render пишет PDF в w (спека 012 FR-2).
func Render(w io.Writer, d Data) error {
	p := fpdf.New("L", "mm", "A4", "")
	p.SetMargins(15, 12, 15)
	p.SetAutoPageBreak(true, 14)
	p.AddUTF8FontFromBytes("dejavu", "", fontRegular)
	p.AddUTF8FontFromBytes("dejavu", "B", fontBold)
	p.AddUTF8FontFromBytes("dejavu", "I", fontOblique)
	p.RegisterImageOptionsReader("emblem", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(emblemPNG))
	p.RegisterImageOptionsReader("watermark", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(watermarkPNG))
	p.SetTextColor(inkR, inkG, inkB)
	p.SetDrawColor(inkR, inkG, inkB)

	// водяной знак на каждой странице (образец: крупная эмблема по центру)
	p.SetHeaderFunc(func() {
		wPage, hPage := p.GetPageSize()
		size := 150.0
		p.ImageOptions("watermark", (wPage-size)/2, (hPage-size)/2, size, size, false,
			fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	})

	r := renderer{p: p}

	num := d.DocNum
	if strings.TrimSpace(num) == "" {
		num = "б/н"
	}
	date := d.DocDate.Format("02.01.2006")

	// 1. Казахское уведомление
	r.titlePage(d.RecipientKZ, "", textKZ)
	// 2. Казахское приложение
	r.appendixHeader(fmt.Sprintf("Қазақстан Республикасы Қаржы министрлігі\nІшкі мемлекеттік аудит комитетінің\n№%s хабарламасына қосымша", num))
	r.table(d.Rows, [10]string{"№\nр/с", "Жөнелтушінің\nатауы", "Төлем нөмірі", "Төлем күні", "Тегі", "Аты", "Әкесінің\nаты", "ЖСН", "Карт-шот", "Сомасы"}, "Алушының мәліметтері")
	r.footer(fmt.Sprintf("%s,\nҚол койған: %s %s\n\nОрын. %s, тел %s, эл.адрес %s\nХабарлама №:%s %s жіберілді",
		d.DeptTitle, orDash(d.SignerPosKZ), orDash(d.SignerName), d.ExecName, d.ExecPhone, d.ExecEmail, num, date))
	// 3. Русское уведомление
	recipRU := d.RecipientRU
	if d.RecipientCode != "" {
		recipRU += " - " + d.RecipientCode
	}
	r.titlePage(recipRU, "", textRU)
	// 4. Русское приложение
	r.appendixHeader(fmt.Sprintf("Приложение к уведомлению №%s\nКомитета внутреннего государственного аудита\nМинистерства финансов Республики Казахстан", num))
	r.table(d.Rows, [10]string{"№\nп/п", "Код ГУ\nотправителя", "Номер платежа", "Дата\nплатежа", "Фамилия", "Имя", "Отчество", "ИИН", "Карт-счет", "Сумма"}, "Сведения получателя")
	r.footer(fmt.Sprintf("%s,\nПодписано: %s %s\n\nИсп. %s, тел %s, эл.адрес %s\nУведомление №:%s от %s",
		d.DeptTitle, orDash(d.SignerPosition), orDash(d.SignerName), d.ExecName, d.ExecPhone, d.ExecEmail, num, date))

	return p.Output(w)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// titlePage — титульная страница языковой части: шапка с эмблемой, адресат, текст.
func (r *renderer) titlePage(recipient, _ string, body string) {
	p := r.p
	p.AddPage()
	wPage, _ := p.GetPageSize()
	usable := wPage - 30 // поля 15+15

	// шапка: каз слева, эмблема в центре, рус справа (жирный)
	colW := (usable - 40) / 2
	top := p.GetY()
	p.SetFont("dejavu", "B", 13)
	p.SetXY(15, top)
	p.MultiCell(colW, 6.5, headKZ, "", "C", false)
	hLeft := p.GetY() - top
	p.ImageOptions("emblem", 15+colW+5, top, 30, 30, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	p.SetXY(15+colW+40, top)
	p.MultiCell(colW, 6.5, headRU, "", "C", false)
	if p.GetY()-top < hLeft {
		p.SetY(top + hLeft)
	}
	p.Ln(14)

	// адресат справа
	p.SetFont("dejavu", "", 13)
	p.SetX(wPage/2 - 15)
	p.MultiCell(usable/2+15, 6.5, recipient, "", "R", false)
	p.Ln(8)

	// текст: абзацы с красной строкой, ключевые обороты не выделяем жирным программно —
	// весь блок одним начертанием (допустимое упрощение, текст идентичен образцу)
	p.SetFont("dejavu", "", 13)
	for _, para := range strings.Split(body, "\n") {
		p.SetX(15)
		p.MultiCell(usable, 6.8, "        "+para, "", "J", false)
	}
}

// appendixHeader — новая страница приложения с заголовком справа.
func (r *renderer) appendixHeader(title string) {
	p := r.p
	p.AddPage()
	wPage, _ := p.GetPageSize()
	p.SetFont("dejavu", "", 12)
	p.SetX(wPage / 2)
	p.MultiCell(wPage/2-15, 6, title, "", "R", false)
	p.Ln(6)
}

// widths: № | ГУ | платёж | дата | фамилия | имя | отчество | ИИН | карт-счёт | сумма
var colW = [10]float64{10, 24, 42, 24, 32, 26, 28, 30, 26, 25}

// table — таблица приложения с групповым заголовком «Сведения получателя» (образец).
func (r *renderer) table(rows []Row, heads [10]string, groupTitle string) {
	p := r.p
	p.SetFont("dejavu", "", 9)

	drawHeader := func() {
		x0 := 15.0
		p.SetX(x0)
		p.SetFont("dejavu", "", 9)
		topH, botH := 8.0, 12.0
		y := p.GetY()
		// первая строка шапки: 4 колонки + объединённая «Сведения получателя»
		x := x0
		for i := 0; i < 4; i++ {
			p.Rect(x, y, colW[i], topH+botH, "D")
			cellTextCentered(p, x, y, colW[i], topH+botH, heads[i])
			x += colW[i]
		}
		groupW := 0.0
		for i := 4; i < 10; i++ {
			groupW += colW[i]
		}
		p.Rect(x, y, groupW, topH, "D")
		cellTextCentered(p, x, y, groupW, topH, groupTitle)
		// вторая строка: подколонки получателя
		for i := 4; i < 10; i++ {
			p.Rect(x, y+topH, colW[i], botH, "D")
			cellTextCentered(p, x, y+topH, colW[i], botH, heads[i])
			x += colW[i]
		}
		p.SetY(y + topH + botH)
	}

	drawHeader()
	for i, row := range rows {
		cells := [10]string{
			fmt.Sprintf("%d", i+1), row.GU, row.PpoPp, fmtDate(row.PaymentDate),
			row.FM, row.NM, row.FT, row.IIN, row.LA1, row.AmountPart,
		}
		// высота строки — по самой «высокой» ячейке
		p.SetFont("dejavu", "", 9)
		maxLines := 1
		for j, c := range cells {
			n := len(p.SplitText(c, colW[j]-2))
			if n > maxLines {
				maxLines = n
			}
		}
		h := float64(maxLines)*4.2 + 2.4

		_, hPage := p.GetPageSize()
		if p.GetY()+h > hPage-16 {
			p.AddPage()
			drawHeader()
		}
		x := 15.0
		y := p.GetY()
		for j, c := range cells {
			p.Rect(x, y, colW[j], h, "D")
			p.SetXY(x+1, y+1.2)
			p.MultiCell(colW[j]-2, 4.2, c, "", "L", false)
			x += colW[j]
		}
		p.SetY(y + h)
	}
}

func cellTextCentered(p *fpdf.Fpdf, x, y, w, h float64, text string) {
	lines := strings.Split(text, "\n")
	lineH := 4.2
	startY := y + (h-float64(len(lines))*lineH)/2
	for i, ln := range lines {
		p.SetXY(x, startY+float64(i)*lineH)
		p.CellFormat(w, lineH, ln, "", 0, "C", false, 0, "")
	}
}

// footer — подвал курсивом после таблицы (образец: с отступом, на текущей странице).
func (r *renderer) footer(text string) {
	p := r.p
	p.Ln(16)
	wPage, hPage := p.GetPageSize()
	if p.GetY() > hPage-50 {
		p.AddPage()
		p.Ln(16)
	}
	p.SetFont("dejavu", "I", 12)
	p.SetX(28)
	p.MultiCell(wPage-56, 6.2, text, "", "L", false)
}

func fmtDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("02.01.2006")
}
