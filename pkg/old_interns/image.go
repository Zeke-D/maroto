package old_interns

import (
	"bytes"
	"encoding/base64"
	"errors"

	"github.com/Zeke-D/maroto/pkg/consts"
	"github.com/Zeke-D/maroto/pkg/old_interns/fpdf"
	"github.com/Zeke-D/maroto/pkg/props"
	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
)

// Image is the abstraction which deals of how to add images in a PDF.
type Image interface {
	AddFromFile(path string, cell Cell, prop props.Rect) (err error)
	AddFromFileAbsolute(path string, cell Cell) error
	AddFromBase64(stringBase64 string, cell Cell, prop props.Rect, extension consts.Extension) (err error)
	AddImageToPdf(imageLabel string, info *gofpdf.ImageInfoType, cell Cell, prop props.Rect)
}

type image struct {
	pdf  fpdf.Fpdf
	math Math
}

// NewImage create an Image.
func NewImage(pdf fpdf.Fpdf, math Math) *image {
	return &image{
		pdf,
		math,
	}
}

// AddFromFile open an image from disk and add to PDF.
func (s *image) AddFromFile(path string, cell Cell, prop props.Rect) error {
	info := s.pdf.RegisterImageOptions(path, gofpdf.ImageOptions{
		ReadDpi:   false,
		ImageType: "",
	})

	if info == nil {
		return errors.New("could not register image options, maybe path/name is wrong")
	}

	s.AddImageToPdf(path, info, cell, prop)
	return nil
}

// adds an image from a file at the given values of cell
func (s *image) AddFromFileAbsolute(path string, cell Cell) error {
	info := s.pdf.RegisterImageOptions(path, gofpdf.ImageOptions{
		ReadDpi:   false,
		ImageType: "",
	})

	if info == nil {
		return errors.New("could not register image options, maybe path/name is wrong")
	}

	s.pdf.Image(path, cell.X, cell.Y, cell.Width, cell.Height, false, "", 0, "")
	return nil
}

// AddFromBase64 use a base64 string to add to PDF.
func (s *image) AddFromBase64(stringBase64 string, cell Cell, prop props.Rect, extension consts.Extension) error {
	imageID, _ := uuid.NewRandom()

	ss, _ := base64.StdEncoding.DecodeString(stringBase64)

	info := s.pdf.RegisterImageOptionsReader(
		imageID.String(),
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(extension),
		},
		bytes.NewReader(ss),
	)

	if info == nil {
		return errors.New("could not register image options, maybe path/name is wrong")
	}

	s.AddImageToPdf(imageID.String(), info, cell, prop)
	return nil
}

func (s *image) AddImageToPdf(imageLabel string, info *gofpdf.ImageInfoType, cell Cell, prop props.Rect) {
	var x, y, w, h float64
	if prop.Center {
		x, y, w, h = s.math.GetRectCenterColProperties(info.Width(), info.Height(), cell.Width, cell.Height, cell.X, prop.Percent)
	} else {
		x, y, w, h = s.math.GetRectNonCenterColProperties(info.Width(), info.Height(), cell.Width, cell.Height, cell.X, prop)
	}
	s.pdf.Image(imageLabel, x, y+cell.Y, w, h, false, "", 0, "")
}
