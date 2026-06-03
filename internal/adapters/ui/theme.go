// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import "github.com/gdamore/tcell/v2"

type ThemeMode string

const (
	ThemeDark  ThemeMode = "dark"
	ThemeLight ThemeMode = "light"
)

type Theme struct {
	Mode ThemeMode

	PrimitiveBackground tcell.Color
	ContrastBackground  tcell.Color
	HeaderBackground    tcell.Color
	InputBackground     tcell.Color
	StatusBackground    tcell.Color
	Border              tcell.Color
	Title               tcell.Color
	PrimaryText         tcell.Color
	SecondaryText       tcell.Color
	TertiaryText        tcell.Color
	Graphics            tcell.Color
	SelectedBackground  tcell.Color
	SelectedText        tcell.Color

	KeyText      string
	AliasText    string
	HostText     string
	MetaText     string
	DimText      string
	TagText      string
	TagBack      string
	Accent       string
	AccentStrong string
	Success      string
	Warning      string
	Error        string
	Separator    string
}

var CurrentTheme = themeForMode(ThemeDark)

func (m ThemeMode) String() string {
	switch m {
	case ThemeDark:
		return "Dark"
	case ThemeLight:
		return "Light"
	default:
		return "Dark"
	}
}

func (m ThemeMode) Toggle() ThemeMode {
	if m == ThemeDark {
		return ThemeLight
	}
	return ThemeDark
}

func (m ThemeMode) valid() bool {
	return m == ThemeDark || m == ThemeLight
}

func SetCurrentTheme(mode ThemeMode) {
	if !mode.valid() {
		mode = ThemeDark
	}
	CurrentTheme = themeForMode(mode)
}

func themeForMode(mode ThemeMode) Theme {
	if mode == ThemeLight {
		return Theme{
			Mode:                ThemeLight,
			PrimitiveBackground: tcell.Color255,
			ContrastBackground:  tcell.Color252,
			HeaderBackground:    tcell.Color253,
			InputBackground:     tcell.Color254,
			StatusBackground:    tcell.Color252,
			Border:              tcell.Color245,
			Title:               tcell.Color238,
			PrimaryText:         tcell.Color235,
			SecondaryText:       tcell.Color240,
			TertiaryText:        tcell.Color243,
			Graphics:            tcell.Color245,
			SelectedBackground:  tcell.Color117,
			SelectedText:        tcell.Color232,
			KeyText:             "#111111",
			AliasText:           "#111111",
			HostText:            "#444444",
			MetaText:            "#666666",
			DimText:             "#777777",
			TagText:             "black",
			TagBack:             "#9AD0FF",
			Accent:              "#0A66C2",
			AccentStrong:        "#075A9E",
			Success:             "#1F8A4C",
			Warning:             "#9A6700",
			Error:               "#C62828",
			Separator:           "#CCCCCC",
		}
	}

	return Theme{
		Mode:                ThemeDark,
		PrimitiveBackground: tcell.Color232,
		ContrastBackground:  tcell.Color235,
		HeaderBackground:    tcell.Color234,
		InputBackground:     tcell.Color233,
		StatusBackground:    tcell.Color235,
		Border:              tcell.Color238,
		Title:               tcell.Color250,
		PrimaryText:         tcell.Color252,
		SecondaryText:       tcell.Color245,
		TertiaryText:        tcell.Color245,
		Graphics:            tcell.Color238,
		SelectedBackground:  tcell.Color24,
		SelectedText:        tcell.Color255,
		KeyText:             "white",
		AliasText:           "white",
		HostText:            "#AAAAAA",
		MetaText:            "#888888",
		DimText:             "#8A8A8A",
		TagText:             "black",
		TagBack:             "#5FAFFF",
		Accent:              "#55AAFF",
		AccentStrong:        "#55D7FF",
		Success:             "#A0FFA0",
		Warning:             "#FFCC66",
		Error:               "#FF6B6B",
		Separator:           "#444444",
	}
}

func colorText(color, text string) string {
	return "[" + color + "]" + text + "[-]"
}

func keyText(text string) string {
	return colorText(CurrentTheme.KeyText, text)
}
