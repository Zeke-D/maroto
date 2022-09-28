package pdf

import (
	"bytes"
	"errors"
	"io"

	"github.com/google/uuid"

	"github.com/Zeke-D/maroto/pkg/color"
	"github.com/Zeke-D/maroto/pkg/old_interns/fpdf"

	"github.com/Zeke-D/maroto/pkg/consts"
	"github.com/Zeke-D/maroto/pkg/old_interns"
	"github.com/Zeke-D/maroto/pkg/props"
	"github.com/jung-kurt/gofpdf"
)

const (
	defaultTopMargin   = 10
	defaultLeftMargin  = 10
	defaultRightMargin = 10
	defaultFontSize    = 16
)

// Maroto is the principal abstraction to create a PDF document.
type Maroto interface {
	// Grid System
	Row(height float64, closure func())
	Col(width uint, closure func())
	ColSpace(gridSize uint)

	// Registers
	RegisterHeader(closure func())
	RegisterFooter(closure func())

	// Outside Col/Row Components
	TableList(header []string, contents [][]string, prop ...props.TableList)
	Line(spaceHeight float64, prop ...props.Line)

	// Inside Col/Row Components
	Text(text string, prop ...props.Text)
	FileImage(filePathName string, prop ...props.Rect) (err error)
	Base64Image(base64 string, extension consts.Extension, prop ...props.Rect) (err error)
	ByteImage(bytesReader io.Reader, extension consts.Extension, prop ...props.Rect) (err error)
	ByteImageAbsolute(bytesReader io.Reader, extension consts.Extension, x float64, y float64, w float64, h float64) error
	Barcode(code string, prop ...props.Barcode) error
	QrCode(code string, prop ...props.Rect)
	DataMatrixCode(code string, prop ...props.Rect)
	Signature(label string, prop ...props.Font)

	// File System
	OutputFileAndClose(filePathName string) error
	Output() (bytes.Buffer, error)

	// Helpers
	AddPage()
	SetBorder(on bool)
	SetBackgroundColor(color color.Color)
	SetAliasNbPages(alias string)
	SetFirstPageNb(number int)
	GetBorder() bool
	GetPageSize() (width float64, height float64)
	GetCurrentPage() int
	GetCurrentOffset() float64
	SetPageMargins(left, top, right float64)
	GetPageMargins() (left float64, top float64, right float64, bottom float64)
	SetCompression(compress bool)

	// Fonts
	AddUTF8Font(familyStr string, styleStr consts.Style, fileStr string)
	AddUTF8FontFromBytes(familyStr string, styleStr consts.Style, utf8Bytes []byte)
	SetFontLocation(fontDirStr string)
	SetProtection(actionFlag byte, userPassStr, ownerPassStr string)
	SetDefaultFontFamily(fontFamily string)
	GetDefaultFontFamily() string

	AsPdfMaroto() PdfMaroto
	SizeBox() props.Box
}

// PdfMaroto is the principal structure which implements Maroto abstraction.
type PdfMaroto struct {
	// Gofpdf wrapper.
	Pdf fpdf.Fpdf

	// Components.
	Math            old_interns.Math
	Font            old_interns.Font
	TextHelper      old_interns.Text
	SignHelper      old_interns.Signature
	Image           old_interns.Image
	Code            old_interns.Code
	TableListHelper old_interns.TableList
	LineHelper      old_interns.Line

	// Closures with Maroto Header and Footer logic.
	headerClosure func()
	footerClosure func()

	// Computed values.
	pageIndex                 int
	OffsetY                   float64
	RowHeight                 float64
	XColOffset                float64
	ColWidth                  float64
	MarginTop                 float64
	firstPageNb               int
	footerHeight              float64
	headerFooterContextActive bool

	// Page configs.
	calculationMode   bool
	backgroundColor   color.Color
	debugMode         bool
	orientation       consts.Orientation
	pageSize          consts.PageSize
	defaultFontFamily string
}

// NewMarotoCustomSize creates a Maroto instance returning a pointer to PdfMaroto
// Receive an Orientation and a PageSize.
// Use if custom page size is needed. Otherwise use NewMaroto() shorthand if using page sizes from consts.Pagesize.
// If using custom width and height, pageSize is just a string value for the format and takes no effect.
// Width and height inputs are measurements of the page in Portrait orientation.
func NewMarotoCustomSize(orientation consts.Orientation, pageSize consts.PageSize, unitStr string, width, height float64, fontDirStr string) Maroto {
	fpdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: string(orientation),
		UnitStr:        unitStr,
		SizeStr:        string(pageSize),
		Size: gofpdf.SizeType{
			Wd: width,
			Ht: height,
		},
		FontDirStr: fontDirStr,
	})
	fpdf.SetMargins(defaultLeftMargin, defaultTopMargin, defaultRightMargin)

	math := old_interns.NewMath(fpdf)
	font := old_interns.NewFont(fpdf, defaultFontSize, consts.Arial, consts.Bold)
	text := old_interns.NewText(fpdf, math, font)

	signature := old_interns.NewSignature(fpdf, math, text)

	image := old_interns.NewImage(fpdf, math)

	code := old_interns.NewCode(fpdf, math)

	tableList := old_interns.NewTableList(text, font)

	lineHelper := old_interns.NewLine(fpdf)

	maroto := &PdfMaroto{
		Pdf:               fpdf,
		Math:              math,
		Font:              font,
		TextHelper:        text,
		SignHelper:        signature,
		Image:             image,
		Code:              code,
		TableListHelper:   tableList,
		LineHelper:        lineHelper,
		pageSize:          pageSize,
		orientation:       orientation,
		calculationMode:   false,
		backgroundColor:   color.NewWhite(),
		defaultFontFamily: consts.Arial,
	}

	maroto.TableListHelper.BindGrid(maroto)

	maroto.Font.SetFamily(consts.Arial)
	maroto.Font.SetStyle(consts.Bold)
	maroto.Font.SetSize(defaultFontSize)
	maroto.debugMode = false

	maroto.Pdf.AddPage()

	return maroto
}

// NewMaroto create a Maroto instance returning a pointer to PdfMaroto
// Receive an Orientation and a PageSize.
// Shorthand when using a preset page size from consts.PageSize.
func NewMaroto(orientation consts.Orientation, pageSize consts.PageSize) Maroto {
	return NewMarotoCustomSize(orientation, pageSize, "mm", 0, 0, "")
}

func NewMarotoWithFontDir(orientation consts.Orientation, pageSize consts.PageSize, fontDirStr string) Maroto {
	return NewMarotoCustomSize(orientation, pageSize, "mm", 0, 0, fontDirStr)
}

func (pdf *PdfMaroto) SizeBox() props.Box {
	return props.Box{
		X: pdf.XColOffset,
		Y: pdf.OffsetY,
		W: pdf.ColWidth,
		H: pdf.RowHeight,
	}
}

func (pdf *PdfMaroto) ByteImage(bytesReader io.Reader, extension consts.Extension, prop ...props.Rect) error {
	rectProp := props.Rect{}
	if len(prop) > 0 {
		rectProp = prop[0]
	}

	rectProp.MakeValid()

	cell := old_interns.Cell{
		X:      pdf.XColOffset,
		Y:      pdf.OffsetY + rectProp.Top,
		Width:  pdf.ColWidth,
		Height: pdf.RowHeight,
	}

	imageID, _ := uuid.NewRandom()

	info := pdf.Pdf.RegisterImageOptionsReader(
		imageID.String(),
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(extension),
		},
		bytesReader,
	)

	if info == nil {
		return errors.New("could not register image options, maybe path/name is wrong")
	}

	pdf.Image.AddImageToPdf(imageID.String(), info, cell, rectProp)
	return nil
}

func (pdf *PdfMaroto) ByteImageAbsolute(bytesReader io.Reader, extension consts.Extension, x float64, y float64, w float64, h float64) error {
	imageID, _ := uuid.NewRandom()

	info := pdf.Pdf.RegisterImageOptionsReader(
		imageID.String(),
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(extension),
		},
		bytesReader,
	)

	if info == nil {
		return errors.New("ByteImageAbsolute: could not register image options, maybe path/name is wrong")
	}

	pdf.Pdf.Image(imageID.String(), x, y, w, h, false, "", 0, "")
	return nil
}
func (s *PdfMaroto) AsPdfMaroto() PdfMaroto {
	return *s
}

// AddPage adds a new page in the PDF.
func (s *PdfMaroto) AddPage() {
	_, pageHeight := s.Pdf.GetPageSize()
	_, top, _, bottom := s.Pdf.GetMargins()

	totalOffsetY := int(s.OffsetY + s.footerHeight)
	maxOffsetPage := int(pageHeight - bottom - top)

	s.Row(float64(maxOffsetPage-totalOffsetY), func() {
		s.ColSpace(uint(consts.MaxGridSum))
	})
}

// RegisterHeader define a sequence of Rows, Lines ou TableLists
// which will be added in every new page.
func (s *PdfMaroto) RegisterHeader(closure func()) {
	s.headerClosure = closure
}

// RegisterFooter define a sequence of Rows, Lines ou TableLists
// which will be added in every new page.
func (s *PdfMaroto) RegisterFooter(closure func()) {
	s.footerClosure = closure

	// calculation mode execute all row flow but
	// only to calculate the sum of heights.
	s.calculationMode = true
	closure()
	s.calculationMode = false
}

// GetCurrentPage obtain the current page index
// this can be used inside a RegisterFooter/RegisterHeader
// to draw the current page, or to another purposes.
func (s *PdfMaroto) GetCurrentPage() int {
	return s.pageIndex + s.firstPageNb
}

// GetCurrentOffset obtain the current offset in y axis.
func (s *PdfMaroto) GetCurrentOffset() float64 {
	return s.OffsetY
}

// SetPageMargins overrides default margins (10,10,10)
// the new page margin will affect all PDF pages.
func (s *PdfMaroto) SetPageMargins(left, top, right float64) {
	if top > defaultTopMargin {
		s.MarginTop = top - defaultTopMargin
	}

	s.Pdf.SetMargins(left, defaultTopMargin, right)
}

// GetPageMargins returns the set page margins. Comes in order of Left, Top, Right, Bottom
// Default page margins is left: 10, top: 10, right: 10.
func (s *PdfMaroto) GetPageMargins() (left float64, top float64, right float64, bottom float64) {
	left, top, right, bottom = s.Pdf.GetMargins()
	top += s.MarginTop

	return
}

// Signature add a space for a signature inside a cell,
// the space will have a line and a text below.
func (s *PdfMaroto) Signature(label string, prop ...props.Font) {
	signProp := props.Font{
		Color: color.Color{
			Red:   0,
			Green: 0,
			Blue:  0,
		},
	}
	if len(prop) > 0 {
		signProp = prop[0]
	}

	signProp.MakeValid(s.defaultFontFamily)

	cell := old_interns.Cell{
		X:      s.XColOffset,
		Y:      s.OffsetY,
		Width:  s.ColWidth,
		Height: s.RowHeight,
	}

	s.SignHelper.AddSpaceFor(label, cell, signProp.ToTextProp(consts.Center, 0.0, false, 0))
}

// TableList create a table with multiple rows and columns,
// so is not possible use this component inside a row or
// inside a column.
// Headers define the amount of columns from each row.
// Headers have bold style, and localized at the top of table.
// Contents are array of arrays. Each array is one line.
func (s *PdfMaroto) TableList(header []string, contents [][]string, prop ...props.TableList) {
	s.TableListHelper.Create(header, contents, s.defaultFontFamily, prop...)
}

// SetBorder enable the draw of lines in every cell.
// Draw borders in all columns created.
func (s *PdfMaroto) SetBorder(on bool) {
	s.debugMode = on
}

// SetBackgroundColor define the background color of the PDF.
// This method can be used to toggle background from rows.
func (s *PdfMaroto) SetBackgroundColor(color color.Color) {
	s.backgroundColor = color
	s.Pdf.SetFillColor(s.backgroundColor.Red, s.backgroundColor.Green, s.backgroundColor.Blue)
}

// SetFirstPageNb define first page number
// Default: 0.
func (s *PdfMaroto) SetFirstPageNb(number int) {
	s.firstPageNb = number
}

// SetAliasNbPages Defines an alias for the total number of pages.
// It will be substituted as the document is closed.
func (s *PdfMaroto) SetAliasNbPages(alias string) {
	s.Pdf.AliasNbPages(alias)
}

// SetCompression allows to set/unset compression for a page
// Compression is on by default.
func (s *PdfMaroto) SetCompression(compress bool) {
	s.Pdf.SetCompression(compress)
}

// GetBorder return the actual border value.
func (s *PdfMaroto) GetBorder() bool {
	return s.debugMode
}

// GetPageSize return the actual page size.
func (s *PdfMaroto) GetPageSize() (width float64, height float64) {
	return s.Pdf.GetPageSize()
}

// Line draw a line from margin left to margin right
// in the current row.
func (s *PdfMaroto) Line(spaceHeight float64, prop ...props.Line) {
	lineProp := props.Line{
		Color: color.NewBlack(),
	}
	if len(prop) > 0 {
		lineProp = prop[0]
	}
	lineProp.MakeValid(spaceHeight)

	s.Row(spaceHeight, func() {
		s.Col(0, func() {
			width, _ := s.Pdf.GetPageSize()
			left, top, right, _ := s.Pdf.GetMargins()

			const divisorToGetHalf = 2.0
			cell := old_interns.Cell{
				X:      left,
				Y:      s.OffsetY + top + (spaceHeight / divisorToGetHalf),
				Width:  width - right,
				Height: s.OffsetY + top + (spaceHeight / divisorToGetHalf),
			}

			s.LineHelper.Draw(cell, lineProp)
		})
	})
}

// Row define a row and enable add columns inside the row.
// Maroto do not support recursive rows or rows inside columns.

func (s *PdfMaroto) Row(height float64, closure func()) {
	// Used to calculate the height of the footer.
	if s.calculationMode {
		s.footerHeight += height
		return
	}

	_, pageHeight := s.Pdf.GetPageSize()
	_, top, _, bottom := s.Pdf.GetMargins()

	totalOffsetY := int(s.OffsetY + height + s.footerHeight)
	maxOffsetPage := int(pageHeight - bottom - top)

	// Note: The headerFooterContextActive is needed to avoid recursive
	// calls without end, because footerClosure and headerClosure actually
	// have Row calls too.

	// If the new cell to be added pass the useful space counting the
	// height of the footer, add the footer.
	if totalOffsetY > maxOffsetPage {
		if !s.headerFooterContextActive {
			s.headerFooterContextActive = true
			s.footer()
			s.headerFooterContextActive = false
			s.OffsetY = 0
			s.pageIndex++
		}
	}

	// If is a new page, add the header.
	if !s.headerFooterContextActive {
		if s.OffsetY == 0 {
			s.headerFooterContextActive = true
			s.header()
			s.headerFooterContextActive = false
		}
	}

	s.RowHeight = height
	s.XColOffset = 0

	// This closure has the Cols to be executed.
	closure()

	s.OffsetY += s.RowHeight
	s.Pdf.Ln(s.RowHeight)
}

// Col create a column inside a row and enable to add
// components inside. Maroto do not support recursive
// columns or rows inside columns.
func (s *PdfMaroto) Col(width uint, closure func()) {
	if width == 0 {
		width = uint(consts.MaxGridSum)
	}

	percent := float64(width) / consts.MaxGridSum

	pageWidth, _ := s.Pdf.GetPageSize()
	left, _, right, _ := s.Pdf.GetMargins()
	widthPerCol := (pageWidth - right - left) * percent

	s.ColWidth = widthPerCol
	s.createColSpace(widthPerCol)

	// This closure has the components to be executed.
	closure()

	s.XColOffset += s.ColWidth
}

// ColSpace create an empty column inside a row.
func (s *PdfMaroto) ColSpace(gridSize uint) {
	s.Col(gridSize, func() {})
}

// Text create a text inside a cell.
func (s *PdfMaroto) Text(text string, prop ...props.Text) {
	textProp := props.Text{
		Color: color.Color{
			Red:   0,
			Green: 0,
			Blue:  0,
		},
	}

	if len(prop) > 0 {
		textProp = prop[0]
	}

	if textProp.Top > s.RowHeight {
		textProp.Top = s.RowHeight
	}

	cell := old_interns.Cell{
		X:      s.XColOffset + textProp.Left,
		Y:      s.OffsetY + textProp.Top,
		Width:  s.ColWidth,
		Height: 0,
	}

	s.TextCustomCell(text, cell, textProp)

}

func (s *PdfMaroto) TextCustomCell(text string, cell old_interns.Cell, textProp props.Text) {
	textProp.MakeValid(s.defaultFontFamily)
	s.TextHelper.Add(text, cell, textProp)
}

func (s *PdfMaroto) TextAbsolute(text string, textProp props.Text) {
	textProp.MakeValid(s.defaultFontFamily)
	s.TextHelper.AddAbsolute(text, textProp)
}

// FileImage add an Image reading from disk inside a cell.
// Defining Image properties.
func (s *PdfMaroto) FileImage(filePathName string, prop ...props.Rect) error {
	rectProp := props.Rect{}
	if len(prop) > 0 {
		rectProp = prop[0]
	}

	rectProp.MakeValid()

	cell := old_interns.Cell{
		X:      s.XColOffset,
		Y:      s.OffsetY + rectProp.Top,
		Width:  s.ColWidth,
		Height: s.RowHeight,
	}

	return s.Image.AddFromFile(filePathName, cell, rectProp)
}

// Base64Image add an Image reading byte slices inside a cell.
// Defining Image properties.
func (s *PdfMaroto) Base64Image(base64 string, extension consts.Extension, prop ...props.Rect) error {
	rectProp := props.Rect{}
	if len(prop) > 0 {
		rectProp = prop[0]
	}

	rectProp.MakeValid()

	cell := old_interns.Cell{
		X:      s.XColOffset,
		Y:      s.OffsetY + rectProp.Top,
		Width:  s.ColWidth,
		Height: s.RowHeight,
	}

	return s.Image.AddFromBase64(base64, cell, rectProp, extension)
}

// Barcode create an barcode inside a cell.
func (s *PdfMaroto) Barcode(code string, prop ...props.Barcode) (err error) {
	barcodeProp := props.Barcode{}
	if len(prop) > 0 {
		barcodeProp = prop[0]
	}

	barcodeProp.MakeValid()

	cell := old_interns.Cell{
		X:      s.XColOffset,
		Y:      s.OffsetY + barcodeProp.Top,
		Width:  s.ColWidth,
		Height: s.RowHeight,
	}

	err = s.Code.AddBar(code, cell, barcodeProp)

	return
}

// DataMatrixCode creates an datamatrix code inside a cell.
func (s *PdfMaroto) DataMatrixCode(code string, prop ...props.Rect) {
	rectProp := props.Rect{}
	if len(prop) > 0 {
		rectProp = prop[0]
	}
	rectProp.MakeValid()

	cell := old_interns.Cell{
		X:      s.XColOffset,
		Y:      s.OffsetY + rectProp.Top,
		Width:  s.ColWidth,
		Height: s.RowHeight,
	}

	s.Code.AddDataMatrix(code, cell, rectProp)
}

// QrCode create a qrcode inside a cell.
func (s *PdfMaroto) QrCode(code string, prop ...props.Rect) {
	rectProp := props.Rect{}
	if len(prop) > 0 {
		rectProp = prop[0]
	}

	rectProp.MakeValid()

	cell := old_interns.Cell{
		X:      s.XColOffset,
		Y:      s.OffsetY + rectProp.Top,
		Width:  s.ColWidth,
		Height: s.RowHeight,
	}

	s.Code.AddQr(code, cell, rectProp)
}

// OutputFileAndClose save pdf in disk.
func (s *PdfMaroto) OutputFileAndClose(filePathName string) (err error) {
	s.drawLastFooter()
	err = s.Pdf.OutputFileAndClose(filePathName)

	return
}

// Output extract PDF in byte slices.
func (s *PdfMaroto) Output() (bytes.Buffer, error) {
	s.drawLastFooter()
	var buffer bytes.Buffer
	err := s.Pdf.Output(&buffer)
	return buffer, err
}

func (s *PdfMaroto) AddFont(familyStr string, styleStr consts.Style, fileStr string) {
	s.Pdf.AddFont(familyStr, string(styleStr), fileStr)
}

// AddUTF8Font add a custom utf8 font. familyStr is the name of the custom font registered in maroto.
// styleStr is the style of the font and fileStr is the path to the .ttf file.
func (s *PdfMaroto) AddUTF8Font(familyStr string, styleStr consts.Style, fileStr string) {
	s.Pdf.AddUTF8Font(familyStr, string(styleStr), fileStr)
}

func (s *PdfMaroto) AddUTF8FontFromBytes(familyStr string, styleStr consts.Style, utf8Bytes []byte) {
	s.Pdf.AddUTF8FontFromBytes(familyStr, string(styleStr), utf8Bytes)
}

func (s *PdfMaroto) Rect(x float64, y float64, w float64, h float64, fillColor color.Color, borderColor color.Color, borderWidth float64) {
	s.Pdf.SetFillColor(fillColor.Red, fillColor.Green, fillColor.Blue)
	s.Pdf.SetDrawColor(borderColor.Red, borderColor.Green, borderColor.Blue)
	s.Pdf.SetLineWidth(borderWidth)
	s.Pdf.Rect(x, y, w, h, "FD")
}

// SetFontLocation allows you to change the fonts lookup location.  fontDirStr is an absolute path where the fonts should be located.
func (s *PdfMaroto) SetFontLocation(fontDirStr string) {
	s.Pdf.SetFontLocation(fontDirStr)
}

// SetProtection define a password to open the pdf.
func (s *PdfMaroto) SetProtection(actionFlag byte, userPassStr, ownerPassStr string) {
	s.Pdf.SetProtection(actionFlag, userPassStr, ownerPassStr)
}

// SetDefaultFontFamily allows you to customize the default font. By default Arial is the original value.
func (s *PdfMaroto) SetDefaultFontFamily(fontFamily string) {
	s.defaultFontFamily = fontFamily
}

// GetDefaultFontFamily allows you to get the current default font family.
func (s *PdfMaroto) GetDefaultFontFamily() string {
	return s.defaultFontFamily
}

func (s *PdfMaroto) createColSpace(actualWidthPerCol float64) {
	border := ""

	if s.debugMode {
		border = "1"
	}

	s.Pdf.CellFormat(actualWidthPerCol, s.RowHeight, "", border, 0, "C", !s.backgroundColor.IsWhite(), 0, "")
}

func (s *PdfMaroto) drawLastFooter() {
	if s.footerClosure != nil {
		_, pageHeight := s.Pdf.GetPageSize()
		_, top, _, bottom := s.Pdf.GetMargins()

		if s.OffsetY+s.footerHeight < pageHeight-bottom-top {
			totalOffsetY := int(s.OffsetY + s.footerHeight)
			maxOffsetPage := int(pageHeight - bottom - top)

			s.Row(float64(maxOffsetPage-totalOffsetY), func() {
				s.ColSpace(12)
			})

			s.headerFooterContextActive = true
			s.footerClosure()
			s.headerFooterContextActive = false
		}
	}
}

func (s *PdfMaroto) footer() {
	backgroundColor := s.backgroundColor
	s.SetBackgroundColor(color.NewWhite())

	_, pageHeight := s.Pdf.GetPageSize()
	_, top, _, bottom := s.Pdf.GetMargins()

	totalOffsetY := int(s.OffsetY + s.footerHeight)
	maxOffsetPage := int(pageHeight - bottom - top)

	s.Row(float64(maxOffsetPage-totalOffsetY), func() {
		s.ColSpace(uint(consts.MaxGridSum))
	})

	if s.footerClosure != nil {
		s.footerClosure()
	}

	s.SetBackgroundColor(backgroundColor)
}

func (s *PdfMaroto) header() {
	backgroundColor := s.backgroundColor
	s.SetBackgroundColor(color.NewWhite())

	s.Row(s.MarginTop, func() {
		s.ColSpace(uint(consts.MaxGridSum))
	})

	if s.headerClosure != nil {
		s.headerClosure()
	}

	s.SetBackgroundColor(backgroundColor)
}
