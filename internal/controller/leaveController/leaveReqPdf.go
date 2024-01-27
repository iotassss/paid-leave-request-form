// TODO: 定数のケースを整える
package leaveController

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/signintech/gopdf"
)

const (
	PaidLeave         = 1
	SpecialLeave      = 2
	CompensatoryLeave = 3
	CondolenceLeave   = 4

	// A4の幅は210mm = 595pt
	pageWidth = 595.0
	// 文字とアンダーラインの間隔
	underlineOffset = 5.0
	// テキストを表示するコンテナの幅を定義
	containerMarginX = 60.0
	// 文字の高さを設定
	lineHeight = 20.0
)

var departmentMap = map[string]string{
	"1": "GIZTECHPRO事業部",
	"2": "営業部",
	"3": "経理部",
}

var weekdayMap = map[time.Weekday]string{
	time.Sunday:    "日",
	time.Monday:    "月",
	time.Tuesday:   "火",
	time.Wednesday: "水",
	time.Thursday:  "木",
	time.Friday:    "金",
	time.Saturday:  "土",
}

// TODO: 手動でソースコードの設定が必要なので要改善
// システムが大規模化したら、DBに移行して管理ツールで設定できるようにする
// 祝日・会社指定の公休日
var offdays = map[time.Time]bool{
	time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC):  true,
	time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC):  true,
	time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC):  true,
	time.Date(2024, 2, 21, 0, 0, 0, 0, time.UTC): true,
	time.Date(2024, 2, 22, 0, 0, 0, 0, time.UTC): true,
}

// 休暇届設定用のPDF
type leaveReqPdf struct {
	gopdf.GoPdf
}

// pdfオブジェクトを受け取り、pdfオブジェクトの初期化を行う関数
func (pdf *leaveReqPdf) initPDF() {
	// PDFドキュメントを作成
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	// フォントを追加
	fontPath := "./ipaexg.ttf"
	pdf.AddTTFFont("ipaexm", fontPath)
}

// 指定の回数改行
func (pdf *leaveReqPdf) br(lineHeight float64, count int) {
	for i := 0; i < count; i++ {
		pdf.Br(lineHeight)
	}
}

func (pdf *leaveReqPdf) textWithUnderline(c *gin.Context, text string) {
	// テキストの幅を測定
	textWidth, err := pdf.MeasureTextWidth(text)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	pdf.Text(text)

	// テキストの幅に基づいてアンダーラインのX座標を計算
	x := pdf.GetX() - textWidth
	y := pdf.GetY()
	pdf.Line(x, y+underlineOffset, x+textWidth, y+underlineOffset)
}

// Title: _text_ のような形式でテキストを表示する
func (pdf *leaveReqPdf) makeLineComponent(c *gin.Context, title string, text string, width float64) {
	blankBeforeText := 10.0

	pdf.Text(title)

	x := pdf.GetX() + blankBeforeText
	pdf.SetX(x)

	pdf.Text(text)

	// textnの幅を取得
	textWidth, err := pdf.MeasureTextWidth(text)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	x = pdf.GetX() - (blankBeforeText + textWidth)
	y := pdf.GetY()

	if width == -1 {
		endOfContainer := pageWidth - containerMarginX
		pdf.Line(x, y+underlineOffset, endOfContainer, y+underlineOffset)
	} else {
		pdf.Line(x, y+underlineOffset, x+width, y+underlineOffset)
	}
}

// formatPeriod formats the period from start to end dates to the specified format.
func formatPeriod(start, end time.Time) string {
	const outputFormat = "1月 2日" // 出力する日付のフォーマット（曜日は後で追加）

	// 曜日を取得
	startWeekday := weekdayMap[start.Weekday()]
	endWeekday := weekdayMap[end.Weekday()]

	// 期間が何日間か計算
	duration := end.Sub(start) + 24*time.Hour
	days := int(duration.Hours() / 24)

	// フォーマットして結果を返す
	return fmt.Sprintf("            %s ( %s )       〜       %s ( %s )         %d日間",
		start.Format(outputFormat),
		startWeekday,
		end.Format(outputFormat),
		endWeekday,
		days,
	)
}

// GroupPeriodsExcludingDates は指定された期間の開始日・終了日の中から、特定の日付を取り除き、
// 連続する期間の開始日・終了日のmapのスライスを返却します。
func GroupPeriodsExcludingDates(start, end time.Time, excludeDates map[time.Time]bool) ([]map[string]time.Time, error) {
	// 結果を格納するスライス
	var periods []map[string]time.Time

	// 現在の開始日を保持する変数
	currentStart := start

	// 開始日から1日ずつ進めていき、終了日まで処理
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		// 現在の日付が除外リストに含まれるか確認
		if excludeDates[d] {
			// 現在の開始日から1日前までを期間として追加する
			if currentStart.Before(d) {
				periods = append(periods, map[string]time.Time{
					"start": currentStart,
					"end":   d.AddDate(0, 0, -1),
				})
			}
			// 次の期間の開始日を設定
			currentStart = d.AddDate(0, 0, 1)
		}
	}

	// 最後の期間を追加
	if currentStart.Before(end.AddDate(0, 0, 1)) {
		periods = append(periods, map[string]time.Time{
			"start": currentStart,
			"end":   end,
		})
	}

	return periods, nil
}

func getWeekends(start, end time.Time) map[time.Time]bool {
	// 土日のmapを作成
	weekends := map[time.Time]bool{}
	// 土日のmapに土日を追加
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			weekends[d] = true
		}
	}
	return weekends
}

// []map[string]time.Timeのをそれぞれ調べて、月またぎがあれば、月またぎで分割する
func splitByMonth(periods []map[string]time.Time) ([]map[string]time.Time, error) {
	dividedPeriods := []map[string]time.Time{}

	// 月またぎがあるかどうかを調べる
	for _, period := range periods {
		// デバッグ用ログとしてperiodを出力
		fmt.Println(period)
		if period["start"].Month() != period["end"].Month() {
			dividedPeriods = append(dividedPeriods, map[string]time.Time{
				"start": period["start"],
				// period["start"]の翌月の1日の前日を取得
				"end": time.Date(period["start"].Year(), period["start"].Month()+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1),
			})
			dividedPeriods = append(dividedPeriods, map[string]time.Time{
				// period["start"]の月の翌月の1日を取得
				"start": time.Date(period["start"].Year(), period["start"].Month()+1, 1, 0, 0, 0, 0, time.UTC),
				"end":   period["end"],
			})
		} else {
			dividedPeriods = append(dividedPeriods, period)
		}
	}

	if len(dividedPeriods) != len(periods) {
		// 月またぎがあった場合、再帰的に分割する
		return splitByMonth(dividedPeriods)
	}

	return dividedPeriods, nil
}

// 入力された期間を公休もしくは月またぎで分割し、分割した期間を返す関数
func splitPeriod(startStr, endStr string) ([]map[string]time.Time, error) {
	// startStrとendStrをtime.Time型に変換し、startとendに代入
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return nil, err
	}

	// startとendの日付を含む期間の中の土日を取得
	weekends := getWeekends(start, end)

	// offdaysにweekendsを追加
	for k, v := range weekends {
		offdays[k] = v
	}

	// startとendの日付を含む期間の中の土日と公休を取り除いた期間のmapのスライスを取得
	workdays, _ := GroupPeriodsExcludingDates(start, end, offdays)

	// 月またぎがあるかどうかを調べる
	periods, _ := splitByMonth(workdays)

	return periods, nil
}

func (pdf *leaveReqPdf) setLeaveRequestPDF(c *gin.Context, period map[string]time.Time) {
	name := c.PostForm("name")
	department := c.PostForm("department")
	position := c.PostForm("position")
	leaveType := c.PostForm("leave_type")
	reason := c.PostForm("reason")

	// タイトルを表示
	TitleFontSize := 24.0
	pdf.SetFont("ipaexm", "", TitleFontSize)
	x := (pageWidth - TitleFontSize) / 2 // 中央揃えのための開始X座標を計算
	y := 80.0                            // Y座標は適宜調整
	pdf.SetX(x)
	pdf.SetY(y)
	pdf.textWithUnderline(c, "休暇届")
	pdf.br(lineHeight, 3)

	// 現在の日付を取得してフォーマット
	currentDate := time.Now().Format("2006年    1月    2日")
	pdf.SetFont("ipaexm", "", 12) // フォントサイズを設定
	// 右寄せに表示
	currentDateWidth, err := pdf.MeasureTextWidth(currentDate)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	x = pageWidth - containerMarginX - currentDateWidth
	pdf.SetX(x)
	pdf.Text(currentDate)
	pdf.br(lineHeight, 3)

	// フォントサイズを設定
	pdf.SetFont("ipaexm", "", 14)

	// 所属を表示
	pdf.SetX(containerMarginX)
	pdf.makeLineComponent(c, "所属: ", departmentMap[department], 280)

	// 役職を表示
	pdf.SetX(400)
	pdf.makeLineComponent(c, "役職: ", position, -1)
	pdf.br(lineHeight, 3)

	// 氏名を表示
	pdf.SetX(containerMarginX)
	pdf.makeLineComponent(c, "氏名: ", name, -1)
	pdf.br(lineHeight, 3)

	leaveTypeInt, err := strconv.Atoi(leaveType)

	// 休暇の種類を表示
	pdf.SetX(containerMarginX)
	pdf.Text("適用:")
	x = 150.0
	y += lineHeight
	pdf.SetX(x)
	// 休暇の種類によって表示するテキストを変える
	if leaveTypeInt == PaidLeave {
		pdf.Text("レ 有給休暇")
	} else {
		pdf.Text("    有給休暇")
	}
	pdf.Line(x, pdf.GetY()+underlineOffset, x+240, pdf.GetY()+underlineOffset)

	x = 300.0
	pdf.SetX(x)
	if leaveTypeInt == SpecialLeave {
		pdf.Text("レ 特別休暇")
	} else {
		pdf.Text("    特別休暇")
	}
	pdf.br(lineHeight, 3)

	x = 150.0
	pdf.SetX(x)
	// pdf.Text("振替休暇")
	if leaveTypeInt == CompensatoryLeave {
		pdf.Text("レ 振替休暇")
	} else {
		pdf.Text("    振替休暇")
	}
	pdf.Line(x, pdf.GetY()+underlineOffset, x+240, pdf.GetY()+underlineOffset)

	x = 300.0
	pdf.SetX(x)
	// pdf.Text("慶弔休暇")
	if leaveTypeInt == CondolenceLeave {
		pdf.Text("レ 慶弔休暇")
	} else {
		pdf.Text("    慶弔休暇")
	}
	pdf.br(lineHeight, 3)

	// 期間をフォーマット
	periodStr := formatPeriod(period["start"], period["end"])

	// 期間を表示
	pdf.SetX(containerMarginX)
	pdf.makeLineComponent(c, "期間: ", periodStr, -1)
	pdf.br(lineHeight, 3)

	// 理由を表示
	pdf.SetX(containerMarginX)
	y = pdf.GetY() - 20
	pdf.Text("  < 理由 >")
	// 理由を表示する箇所を四角い枠で囲む
	// 理由のy座標を取得する
	y = pdf.GetY() + 10
	pdf.RectFromUpperLeftWithStyle(containerMarginX, y-30.0, pageWidth-containerMarginX*2, 150, "D")
	pdf.Line(containerMarginX, y, pageWidth-containerMarginX, y)

	// 枠の中に理由を表示
	pdf.SetX(containerMarginX + 10)
	pdf.SetY(y + 20)
	pdf.Text(reason)

	// 人事の承認印を押す箇所を枠で囲んで表示
	pdf.br(lineHeight, 3)

	// 人事の承認印を押す箇所を枠で囲んで表示
	pdf.SetX(pageWidth - containerMarginX - 127)
	pdf.SetY(696)
	pdf.Text("人事")
	pdf.SetX(pageWidth - containerMarginX - 53)
	pdf.Text("役員")
	pdf.SetX(pageWidth - containerMarginX - 130)
	pdf.RectFromUpperLeftWithStyle(pageWidth-containerMarginX-150, 700-20, 150, 100, "D")
	pdf.Line(pageWidth-containerMarginX-150, 700, pageWidth-containerMarginX, 700)
	pdf.Line(pageWidth-containerMarginX-75, 700-20, pageWidth-containerMarginX-75, 780)
}

func createPDF(content string) ([]byte, error) {
	var buf bytes.Buffer

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()
	if err := pdf.AddTTFFont("ipaexg", "./ipaexg.ttf"); err != nil {
		return nil, err
	}

	if err := pdf.SetFont("ipaexg", "", 14); err != nil {
		return nil, err
	}

	pdf.Cell(nil, content)
	if _, err := pdf.WriteTo(&buf); err != nil { // この行を変更
		return nil, err
	}

	return buf.Bytes(), nil
}

// 複数のpdfオブジェクトを受け取り、zipファイルを作成し、zipファイルのバイト列を返す関数
func createZip(pdfs []*leaveReqPdf, periods []map[string]time.Time, name string) (zipBytes []byte, err error) {
	// zipファイルを作成
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// PDFを生成してZIPに追加
	for i, pdf := range pdfs {
		// ファイル名を設定
		fileName := fmt.Sprintf("%s_%s~%s休暇申請書.pdf", name, periods[i]["start"].Format("0102"), periods[i]["end"].Format("0102"))

		// ZIPに新しいファイルエントリを作成
		f, err := zipWriter.Create(fileName)
		if err != nil {
			return nil, err
		}

		// PDFをメモリ内で出力し、ZIPファイルエントリに書き込む
		pdfBuf := new(bytes.Buffer)
		_, err = pdf.WriteTo(pdfBuf)
		if err != nil {
			return nil, err
		}
		_, err = f.Write(pdfBuf.Bytes())
		if err != nil {
			return nil, err
		}
	}

	// ZIP書き込みを完了
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	// バイト列を返す
	return buf.Bytes(), nil
}

// ZIPファイルとしてクライアントに送信
func sendZip(c *gin.Context, zipBytes []byte) {
	// ZIPファイルとしてクライアントに送信
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=休暇届.zip")
	c.Data(200, "application/zip", zipBytes)
}

// 作成されたPDFが一つならPDFをZIPにせずにそのままクライアントに送信する関数
func sendPDF(c *gin.Context, pdf *leaveReqPdf) {
	// PDFをメモリ内で出力
	_, err := pdf.WriteTo(c.Writer)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// ヘッダーをセットしてブラウザにダウンロードを促す
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename=paid_leave.pdf")
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Expires", "0")
	c.Header("Cache-Control", "must-revalidate")
	c.Header("Pragma", "public")
}
