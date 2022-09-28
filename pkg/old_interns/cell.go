package old_interns

// Cell represents a cell inside the PDF.
type Cell struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

func (c *Cell) AsMm() (copy Cell) {
	pt2mm := 25.4 / 72
	copy.X = c.X * pt2mm
	copy.Y = c.Y * pt2mm
	copy.Width = c.Width * pt2mm
	copy.Height = c.Height * pt2mm
	return
}

func (c *Cell) AsPt() (copy Cell) {
	mm2pt := 72 / 25.4
	copy.X = c.X * mm2pt
	copy.Y = c.Y * mm2pt
	copy.Width = c.Width * mm2pt
	copy.Height = c.Height * mm2pt
	return
}
