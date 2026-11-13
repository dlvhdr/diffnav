package common

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
	tint "github.com/lrstanley/bubbletint/v2"
)

// lipglossColorToHex converts a color.Color to hex string
func LipglossColorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

type Styles struct {
	Tint   *tint.Tint
	Colors Colors
}

type Colors struct {
	SelectionFg       color.Color
	SelectionBg       color.Color
	DarkerSelectionBg color.Color
	FaintBlue         func() color.Color
}

func MakeStyles() Styles {
	t := Themes.Current()
	if t.ID == tint.TintTokyoNightStorm.ID {
		t.BrightGreen = tint.FromHex("#9ece6a")
	}

	selectionFg := lipgloss.Darken(t.Blue, 0.3)
	selectionBg := lipgloss.Darken(t.Blue, 0.6)
	colors := Colors{
		SelectionFg:       selectionFg,
		SelectionBg:       selectionBg,
		DarkerSelectionBg: lipgloss.Darken(selectionBg, 0.3),
		FaintBlue: func() color.Color {
			return lipgloss.Darken(t.Blue, 0.8)
		},
	}

	return Styles{
		Tint:   t,
		Colors: colors,
	}
}

var SupportedThemeToDeltaSyntax = map[string]string{
	tint.TintCatppuccinFrappe.ID:      "Catppuccin Frappe",
	tint.TintCatppuccinMacchiato.ID:   "Catppuccin Macchiato",
	tint.TintCatppuccinMocha.ID:       "Catppuccin Mocha",
	tint.TintCatppuccinLatte.ID:       "Catppuccin Latte",
	tint.TintNeon.ID:                  "DarkNeon",
	tint.TintDracula.ID:               "Dracula",
	tint.TintMonokaiPro.ID:            "Monokai Extended",
	tint.TintNord.ID:                  "Nord",
	tint.TintOneHalfDark.ID:           "OneHalfDark",
	tint.TintGruvboxLight.ID:          "gruvbox-light",
	tint.TintGruvboxDark.ID:           "gruvbox-dark",
	tint.TintTokyoNight.ID:            "tokyonight_night",
	tint.TintZenburn.ID:               "zenburn",
	tint.TintGithub.ID:                "GitHub",
	tint.TintDimmedMonokai.ID:         "Monokai Extended Light",
	tint.TintOneHalfLight.ID:          "OneHalfLight",
	tint.TintBuiltinSolarizedLight.ID: "Solarized (light)",
	tint.TintBuiltinSolarizedDark.ID:  "Solarized (dark)",
}

var Themes *tint.Registry

func RegisterSupportedTints() {
	Themes = tint.NewRegistry(
		tint.TintTokyoNight,
		tint.TintCatppuccinFrappe,
		tint.TintCatppuccinMacchiato,
		tint.TintCatppuccinMocha,
		tint.TintCatppuccinLatte,
		tint.TintNeon,
		tint.TintDracula,
		tint.TintMonokaiPro,
		tint.TintNord,
		tint.TintOneHalfDark,
		tint.TintGruvboxLight,
		tint.TintGruvboxDark,
		tint.TintTokyoNight,
		tint.TintZenburn,
		tint.TintGithub,
		tint.TintDimmedMonokai,
		tint.TintOneHalfLight,
		tint.TintBuiltinSolarizedLight,
		tint.TintBuiltinSolarizedDark,
	)
}
