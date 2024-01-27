// TODO: タイトルが右によっている？
// TODO: HTTPヘッダの設定
package leaveController

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Form(c *gin.Context) {
	c.HTML(http.StatusOK, "form.html", gin.H{
		"title": "Form",
	})
}

// フォームから入力された内容を元に、休暇届の設定をしたPDFを生成し、クライアントに送信する関数
func Generate(c *gin.Context) {
	// フォームから入力された休暇開始日と終了日を取得
	start := c.PostForm("start")
	end := c.PostForm("end")
	name := c.PostForm("name")

	// 期間を分割し、分割した期間を取得
	periods, err := splitPeriod(start, end)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	// 分割した期間それぞれに対応する休暇届の設定をしたPDFを生成
	pdfs := []*leaveReqPdf{}
	for _, period := range periods {
		// PDFを生成
		pdf := leaveReqPdf{}
		pdf.initPDF()
		pdf.setLeaveRequestPDF(c, period)
		// PDFをスライスに追加
		pdfs = append(pdfs, &pdf)
	}

	// PDFのスライスが一つなら、sendPDFを呼び出し、クライアントに送信
	if len(pdfs) == 1 {
		sendPDF(c, pdfs[0])
		return
	}
	// PDFのスライスが複数なら、createZipを呼び出し、ZIPファイルを作成
	zipBytes, err := createZip(pdfs, periods, name)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	// ZIPファイルをクライアントに送信
	sendZip(c, zipBytes)
}
