package service

import (
	"embed"

	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
)

//go:embed fonts/*.ttf
var fontFiles embed.FS

const FontFamily = "Liberation"

func LoadEmbeddedFonts() []*entity.CustomFont {
	fonts := make([]*entity.CustomFont, 0, 4)

	fontMap := map[string]fontstyle.Type{
		"fonts/LiberationSans-Regular.ttf":    fontstyle.Normal,
		"fonts/LiberationSans-Bold.ttf":       fontstyle.Bold,
		"fonts/LiberationSans-Italic.ttf":     fontstyle.Italic,
		"fonts/LiberationSans-BoldItalic.ttf": fontstyle.BoldItalic,
	}

	for path, style := range fontMap {
		data, err := fontFiles.ReadFile(path)
		if err != nil {
			continue
		}
		fonts = append(fonts, &entity.CustomFont{
			Family: FontFamily,
			Style:  style,
			Bytes:  data,
		})
	}

	return fonts
}
