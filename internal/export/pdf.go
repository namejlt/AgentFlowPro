package export

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func BuildPDF(ctx context.Context, markdown string, chromeBin string) ([]byte, error) {
	html := `<!doctype html><html><head><meta charset="utf-8"><style>body{font-family:-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica,Arial;line-height:1.6;padding:24px;}pre{white-space:pre-wrap;}</style></head><body><pre>` +
		escHTML(markdown) + `</pre></body></html>`

	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.NoSandbox, chromedp.DisableGPU)
	if strings.TrimSpace(chromeBin) != "" {
		opts = append(opts, chromedp.ExecPath(chromeBin))
	}
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()
	bctx, cancel2 := chromedp.NewContext(allocCtx)
	defer cancel2()
	bctx, cancel3 := context.WithTimeout(bctx, 90*time.Second)
	defer cancel3()

	data := "data:text/html;charset=utf-8;base64," + base64.StdEncoding.EncodeToString([]byte(html))
	var pdf []byte
	err := chromedp.Run(bctx,
		chromedp.Navigate(data),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdf, _, err = page.PrintToPDF().WithPrintBackground(true).Do(ctx)
			return err
		}),
	)
	return pdf, err
}

func escHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func PDFFilename(title string) string {
	return fmt.Sprintf("%s.pdf", strings.ReplaceAll(strings.TrimSpace(title), "/", "-"))
}
