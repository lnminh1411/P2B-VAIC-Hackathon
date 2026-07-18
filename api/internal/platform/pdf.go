package platform

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/p2b/p2b/internal/domain"
)

type applicationMinutes struct {
	Number          string
	CompanyName     string
	TaxCode         string
	LegalForm       string
	Address         string
	Website         string
	Province        string
	PolicyTitle     string
	PolicyAgency    string
	PolicyVersion   int
	PolicySummary   string
	PolicySourceURL string
	CreatedAt       time.Time
	Overview        string
	ReviewScope     string
	Conclusion      string
	Checklist       []ChecklistItem
}

func newApplicationMinutes(pass domain.Passport, policy domain.Policy, application Application, checklist Checklist) applicationMinutes {
	createdAt := application.UpdatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	return applicationMinutes{
		Number:          strings.ToUpper(application.ID[:min(8, len(application.ID))]),
		CompanyName:     firstNonEmpty(passportField(pass, "legal_name"), pass.CompanyName, "Chưa có dữ liệu"),
		TaxCode:         firstNonEmpty(passportField(pass, "tax_code"), "Chưa có dữ liệu"),
		LegalForm:       firstNonEmpty(passportField(pass, "legal_form"), "Chưa có dữ liệu"),
		Address:         firstNonEmpty(passportField(pass, "registered_address"), "Chưa có dữ liệu"),
		Website:         firstNonEmpty(pass.Website, "Chưa có dữ liệu"),
		Province:        firstNonEmpty(passportField(pass, "province"), "................"),
		PolicyTitle:     policy.Title,
		PolicyAgency:    policy.Agency,
		PolicyVersion:   policy.Version,
		PolicySummary:   firstNonEmpty(policy.Benefit, "Chưa có nội dung trích yếu được xác minh."),
		PolicySourceURL: firstNonEmpty(policy.SourceURL, "Chưa có đường dẫn nguồn"),
		CreatedAt:       createdAt,
		Overview:        application.Sections["company_overview"],
		ReviewScope:     application.Sections["support_need"],
		Conclusion:      application.Sections["proposal"],
		Checklist:       checklist.Items,
	}
}

func renderApplicationMinutes(data applicationMinutes) ([]byte, error) {
	regularFont, boldFont, err := loadPDFFonts()
	if err != nil {
		return nil, err
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 18, 20)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddUTF8FontFromBytes("P2B", "", regularFont)
	pdf.AddUTF8FontFromBytes("P2B", "B", boldFont)
	pdf.SetTitle("Biên bản đối chiếu "+data.PolicyTitle, true)
	pdf.SetAuthor("P2B - Policy to Business", true)
	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-14)
		pdf.SetFont("P2B", "", 8)
		pdf.SetTextColor(90, 90, 90)
		pdf.CellFormat(0, 5, fmt.Sprintf("Mẫu làm việc P2B - Trang %d/{nb}", pdf.PageNo()), "", 0, "C", false, 0, "")
	})
	pdf.AddPage()

	writeAdministrativeHeader(pdf, data)
	writeMinutesBody(pdf, data)
	writeSignatureArea(pdf)

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("render application PDF: %w", err)
	}
	return output.Bytes(), nil
}

func writeAdministrativeHeader(pdf *fpdf.Fpdf, data applicationMinutes) {
	startY := pdf.GetY()
	pdf.SetFont("P2B", "B", 10)
	pdf.SetXY(20, startY)
	pdf.MultiCell(75, 5, strings.ToUpper(data.CompanyName), "", "C", false)
	pdf.SetFont("P2B", "", 9)
	pdf.SetX(20)
	pdf.CellFormat(75, 5, "Số: P2B/"+data.Number, "", 0, "C", false, 0, "")

	pdf.SetXY(105, startY)
	pdf.SetFont("P2B", "B", 10)
	pdf.CellFormat(85, 5, "CỘNG HÒA XÃ HỘI CHỦ NGHĨA VIỆT NAM", "", 2, "C", false, 0, "")
	pdf.SetFont("P2B", "B", 9.5)
	pdf.CellFormat(85, 5, "Độc lập - Tự do - Hạnh phúc", "", 2, "C", false, 0, "")
	pdf.SetFont("P2B", "", 9)
	pdf.CellFormat(85, 4, "----------------", "", 0, "C", false, 0, "")

	pdf.SetY(max(startY+21, pdf.GetY()+5))
	pdf.SetFont("P2B", "", 9.5)
	pdf.CellFormat(0, 5, administrativeDate(data.Province, data.CreatedAt), "", 1, "R", false, 0, "")
	pdf.Ln(5)
	pdf.SetFont("P2B", "B", 15)
	pdf.MultiCell(0, 7, "BIÊN BẢN ĐỐI CHIẾU PHẠM VI VÀ ĐIỀU KIỆN ÁP DỤNG", "", "C", false)
	pdf.SetFont("P2B", "B", 11)
	pdf.MultiCell(0, 6, "Văn bản: "+data.PolicyTitle, "", "C", false)
	pdf.Ln(4)
}

func writeMinutesBody(pdf *fpdf.Fpdf, data applicationMinutes) {
	writeParagraph(pdf, fmt.Sprintf("Hôm nay, %s, tại %s, doanh nghiệp tiến hành lập biên bản đối chiếu thông tin Company Passport với văn bản nêu trên.", spelledOutDate(data.CreatedAt), data.Province))

	writeSectionTitle(pdf, "I. THÔNG TIN DOANH NGHIỆP")
	writeLabelValue(pdf, "Tên doanh nghiệp", data.CompanyName)
	writeLabelValue(pdf, "Mã số thuế/Mã số doanh nghiệp", data.TaxCode)
	writeLabelValue(pdf, "Loại hình pháp lý", data.LegalForm)
	writeLabelValue(pdf, "Địa chỉ trụ sở", data.Address)
	writeLabelValue(pdf, "Website", data.Website)
	writeLabelValue(pdf, "Người đại diện tham gia đối chiếu", "........................................................")

	writeSectionTitle(pdf, "II. CĂN CỨ ĐỐI CHIẾU")
	writeLabelValue(pdf, "Văn bản", data.PolicyTitle)
	writeLabelValue(pdf, "Cơ quan ban hành", firstNonEmpty(data.PolicyAgency, "Chưa có dữ liệu"))
	writeLabelValue(pdf, "Phiên bản dữ liệu P2B", fmt.Sprintf("v%d", data.PolicyVersion))
	writeLabelValue(pdf, "Nguồn chính thức", data.PolicySourceURL)
	writeParagraph(pdf, "Trích yếu/nội dung liên quan: "+data.PolicySummary)

	writeSectionTitle(pdf, "III. NỘI DUNG ĐỐI CHIẾU")
	writeLabelValue(pdf, "Thông tin doanh nghiệp", data.Overview)
	writeLabelValue(pdf, "Phạm vi cần rà soát", data.ReviewScope)
	for index, item := range data.Checklist {
		status := map[string]string{"AVAILABLE": "Đã có", "MISSING": "Còn thiếu", "NEEDS_REVIEW": "Cần rà soát", "NOT_APPLICABLE": "Không áp dụng"}[item.Status]
		if status == "" {
			status = item.Status
		}
		line := fmt.Sprintf("%d. %s - %s", index+1, item.Title, status)
		if item.EvidenceSource != "" {
			line += ". Căn cứ: " + item.EvidenceSource
		}
		writeParagraph(pdf, line)
	}

	if pdf.GetY() > 190 {
		pdf.AddPage()
	}
	writeSectionTitle(pdf, "IV. KẾT LUẬN VÀ XÁC NHẬN")
	writeParagraph(pdf, data.Conclusion)
	writeParagraph(pdf, "Biên bản này là tài liệu làm việc do P2B hỗ trợ tạo từ dữ liệu đã lưu trong Company Passport và văn bản nguồn. Biên bản không thay thế kết luận pháp lý, ý kiến của cơ quan có thẩm quyền hoặc biểu mẫu chính thức dùng để nộp hồ sơ.")
	writeParagraph(pdf, "Biên bản được lập thành 02 bản có nội dung như nhau; các bên đọc lại, thống nhất nội dung và ký xác nhận dưới đây.")
}

func writeSignatureArea(pdf *fpdf.Fpdf) {
	if pdf.GetY() > 235 {
		pdf.AddPage()
	}
	pdf.Ln(8)
	y := pdf.GetY()
	pdf.SetFont("P2B", "B", 10)
	pdf.SetXY(22, y)
	pdf.CellFormat(75, 6, "NGƯỜI LẬP BIÊN BẢN", "", 0, "C", false, 0, "")
	pdf.SetXY(113, y)
	pdf.CellFormat(75, 6, "ĐẠI DIỆN DOANH NGHIỆP", "", 1, "C", false, 0, "")
	pdf.SetFont("P2B", "", 9)
	pdf.SetX(22)
	pdf.CellFormat(75, 5, "(Ký, ghi rõ họ tên)", "", 0, "C", false, 0, "")
	pdf.SetX(113)
	pdf.CellFormat(75, 5, "(Ký, ghi rõ họ tên, đóng dấu nếu có)", "", 1, "C", false, 0, "")
	pdf.Ln(24)
}

func writeSectionTitle(pdf *fpdf.Fpdf, title string) {
	pdf.Ln(3)
	pdf.SetFont("P2B", "B", 10.5)
	pdf.SetTextColor(20, 20, 20)
	pdf.SetFillColor(242, 245, 248)
	pdf.CellFormat(0, 7, title, "", 1, "L", true, 0, "")
	pdf.Ln(1)
}

func writeLabelValue(pdf *fpdf.Fpdf, label, value string) {
	pdf.SetFont("P2B", "B", 9.5)
	pdf.MultiCell(0, 5.5, label+":", "", "L", false)
	pdf.SetFont("P2B", "", 9.5)
	pdf.SetX(25)
	pdf.MultiCell(165, 5.5, firstNonEmpty(strings.TrimSpace(value), "Chưa có dữ liệu"), "", "J", false)
}

func writeParagraph(pdf *fpdf.Fpdf, value string) {
	pdf.SetFont("P2B", "", 9.5)
	pdf.MultiCell(0, 5.5, strings.TrimSpace(value), "", "J", false)
	pdf.Ln(1)
}

func passportField(pass domain.Passport, key string) string {
	field, ok := pass.Fields[key]
	if !ok || field.Value == nil || field.Status != domain.FieldConfirmed {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(field.Value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func administrativeDate(province string, value time.Time) string {
	return fmt.Sprintf("%s, ngày %02d tháng %02d năm %04d", province, value.Day(), value.Month(), value.Year())
}

func spelledOutDate(value time.Time) string {
	return fmt.Sprintf("ngày %02d tháng %02d năm %04d", value.Day(), value.Month(), value.Year())
}

func loadPDFFonts() ([]byte, []byte, error) {
	type fontPair struct{ regular, bold string }
	pairs := []fontPair{
		{os.Getenv("P2B_PDF_FONT_REGULAR"), os.Getenv("P2B_PDF_FONT_BOLD")},
		{"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf", "/usr/share/fonts/truetype/noto/NotoSans-Bold.ttf"},
		{"/System/Library/Fonts/Supplemental/Verdana.ttf", "/System/Library/Fonts/Supplemental/Verdana Bold.ttf"},
	}
	for _, pair := range pairs {
		if pair.regular == "" || pair.bold == "" {
			continue
		}
		regular, regularErr := os.ReadFile(pair.regular)
		bold, boldErr := os.ReadFile(pair.bold)
		if regularErr == nil && boldErr == nil {
			return regular, bold, nil
		}
	}
	return nil, nil, fmt.Errorf("render application PDF: Unicode font files are unavailable")
}
