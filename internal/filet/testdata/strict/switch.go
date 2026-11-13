package fixture

import "net/http"

type Color int

const (
	Red Color = iota
	Green
	Blue
)

const Crimson = Red

type Level string

const (
	Low  Level = "low"
	High Level = "high"
)

type Port int

const HTTPPort Port = 80

func missesBlue(c Color) string {
	switch c { // want
	case Red:
		return "red"
	case Green:
		return "green"
	}
	return ""
}

func coversAll(c Color) string {
	switch c {
	case Crimson, Green:
		return "warm"
	case Blue:
		return "cold"
	}
	return ""
}

func hasDefault(c Color) string {
	switch c {
	case Red:
		return "red"
	default:
		return "other"
	}
}

func missesString(l Level) bool {
	switch l { // want
	case Low:
		return true
	}
	return false
}

func singleConstant(p Port) bool {
	switch p {
	case HTTPPort:
		return true
	}
	return false
}

func importedEnum(s http.SameSite) string {
	switch s { // want
	case http.SameSiteLaxMode:
		return "lax"
	}
	return ""
}

func tagless(c Color) bool {
	switch {
	case c == Red:
		return true
	}
	return false
}

func plainInt(n int) bool {
	switch n {
	case 1:
		return true
	}
	return false
}
