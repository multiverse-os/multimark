package markdown

import (
	"bytes"
	"io"
)

// MultiverseRenderer is a struct containing state of a Multiverse renderer.
type MultiverseRenderer struct {
	inSingleQuote bool
	inDoubleQuote bool
	callbacks     [256]multiverseCallback
}

func wordBoundary(c byte) bool {
	return c == 0 || isspace(c) || ispunct(c)
}

func tolower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c - 'A' + 'a'
	}
	return c
}

func isdigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func multiverseQuoteHelper(out *bytes.Buffer, previousChar byte, nextChar byte, quote byte, isOpen *bool, addNBSP bool) bool {
	// edge of the buffer is likely to be a tag that we don't get to see,
	// so we treat it like text sometimes

	// enumerate all sixteen possibilities for (previousChar, nextChar)
	// each can be one of {0, space, punct, other}
	switch {
	case previousChar == 0 && nextChar == 0:
		// context is not any help here, so toggle
		*isOpen = !*isOpen
	case isspace(previousChar) && nextChar == 0:
		// [ "] might be [ "<code>foo...]
		*isOpen = true
	case ispunct(previousChar) && nextChar == 0:
		// [!"] hmm... could be [Run!"] or [("<code>...]
		*isOpen = false
	case /* isnormal(previousChar) && */ nextChar == 0:
		// [a"] is probably a close
		*isOpen = false
	case previousChar == 0 && isspace(nextChar):
		// [" ] might be [...foo</code>" ]
		*isOpen = false
	case isspace(previousChar) && isspace(nextChar):
		// [ " ] context is not any help here, so toggle
		*isOpen = !*isOpen
	case ispunct(previousChar) && isspace(nextChar):
		// [!" ] is probably a close
		*isOpen = false
	case /* isnormal(previousChar) && */ isspace(nextChar):
		// [a" ] this is one of the easy cases
		*isOpen = false
	case previousChar == 0 && ispunct(nextChar):
		// ["!] hmm... could be ["$1.95] or [</code>"!...]
		*isOpen = false
	case isspace(previousChar) && ispunct(nextChar):
		// [ "!] looks more like [ "$1.95]
		*isOpen = true
	case ispunct(previousChar) && ispunct(nextChar):
		// [!"!] context is not any help here, so toggle
		*isOpen = !*isOpen
	case /* isnormal(previousChar) && */ ispunct(nextChar):
		// [a"!] is probably a close
		*isOpen = false
	case previousChar == 0 /* && isnormal(nextChar) */ :
		// ["a] is probably an open
		*isOpen = true
	case isspace(previousChar) /* && isnormal(nextChar) */ :
		// [ "a] this is one of the easy cases
		*isOpen = true
	case ispunct(previousChar) /* && isnormal(nextChar) */ :
		// [!"a] is probably an open
		*isOpen = true
	default:
		// [a'b] maybe a contraction?
		*isOpen = false
	}

	// Note that with the limited lookahead, this non-breaking
	// space will also be appended to single double quotes.
	if addNBSP && !*isOpen {
		out.WriteString("&nbsp;")
	}

	out.WriteByte('&')
	if *isOpen {
		out.WriteByte('l')
	} else {
		out.WriteByte('r')
	}
	out.WriteByte(quote)
	out.WriteString("quo;")

	if addNBSP && *isOpen {
		out.WriteString("&nbsp;")
	}

	return true
}

func (r *MultiverseRenderer) multiverseSingleQuote(out *bytes.Buffer, previousChar byte, text []byte) int {
	if len(text) >= 2 {
		t1 := tolower(text[1])

		if t1 == '\'' {
			nextChar := byte(0)
			if len(text) >= 3 {
				nextChar = text[2]
			}
			if multiverseQuoteHelper(out, previousChar, nextChar, 'd', &r.inDoubleQuote, false) {
				return 1
			}
		}

		if (t1 == 's' || t1 == 't' || t1 == 'm' || t1 == 'd') && (len(text) < 3 || wordBoundary(text[2])) {
			out.WriteString("&rsquo;")
			return 0
		}

		if len(text) >= 3 {
			t2 := tolower(text[2])

			if ((t1 == 'r' && t2 == 'e') || (t1 == 'l' && t2 == 'l') || (t1 == 'v' && t2 == 'e')) &&
				(len(text) < 4 || wordBoundary(text[3])) {
				out.WriteString("&rsquo;")
				return 0
			}
		}
	}

	nextChar := byte(0)
	if len(text) > 1 {
		nextChar = text[1]
	}
	if multiverseQuoteHelper(out, previousChar, nextChar, 's', &r.inSingleQuote, false) {
		return 0
	}

	out.WriteByte(text[0])
	return 0
}

func (r *MultiverseRenderer) multiverseParens(out *bytes.Buffer, previousChar byte, text []byte) int {
	if len(text) >= 3 {
		t1 := tolower(text[1])
		t2 := tolower(text[2])

		if t1 == 'c' && t2 == ')' {
			out.WriteString("&copy;")
			return 2
		}

		if t1 == 'r' && t2 == ')' {
			out.WriteString("&reg;")
			return 2
		}

		if len(text) >= 4 && t1 == 't' && t2 == 'm' && text[3] == ')' {
			out.WriteString("&trade;")
			return 3
		}
	}

	out.WriteByte(text[0])
	return 0
}

func (r *MultiverseRenderer) multiverseDash(out *bytes.Buffer, previousChar byte, text []byte) int {
	if len(text) >= 2 {
		if text[1] == '-' {
			out.WriteString("&mdash;")
			return 1
		}

		if wordBoundary(previousChar) && wordBoundary(text[1]) {
			out.WriteString("&ndash;")
			return 0
		}
	}

	out.WriteByte(text[0])
	return 0
}

func (r *MultiverseRenderer) multiverseDashLatex(out *bytes.Buffer, previousChar byte, text []byte) int {
	if len(text) >= 3 && text[1] == '-' && text[2] == '-' {
		out.WriteString("&mdash;")
		return 2
	}
	if len(text) >= 2 && text[1] == '-' {
		out.WriteString("&ndash;")
		return 1
	}

	out.WriteByte(text[0])
	return 0
}

func (r *MultiverseRenderer) multiverseAmpVariant(out *bytes.Buffer, previousChar byte, text []byte, quote byte, addNBSP bool) int {
	if bytes.HasPrefix(text, []byte("&quot;")) {
		nextChar := byte(0)
		if len(text) >= 7 {
			nextChar = text[6]
		}
		if multiverseQuoteHelper(out, previousChar, nextChar, quote, &r.inDoubleQuote, addNBSP) {
			return 5
		}
	}

	if bytes.HasPrefix(text, []byte("&#0;")) {
		return 3
	}

	out.WriteByte('&')
	return 0
}

func (r *MultiverseRenderer) multiverseAmp(angledQuotes, addNBSP bool) func(*bytes.Buffer, byte, []byte) int {
	var quote byte = 'd'
	if angledQuotes {
		quote = 'a'
	}

	return func(out *bytes.Buffer, previousChar byte, text []byte) int {
		return r.multiverseAmpVariant(out, previousChar, text, quote, addNBSP)
	}
}

func (r *MultiverseRenderer) multiversePeriod(out *bytes.Buffer, previousChar byte, text []byte) int {
	if len(text) >= 3 && text[1] == '.' && text[2] == '.' {
		out.WriteString("&hellip;")
		return 2
	}

	if len(text) >= 5 && text[1] == ' ' && text[2] == '.' && text[3] == ' ' && text[4] == '.' {
		out.WriteString("&hellip;")
		return 4
	}

	out.WriteByte(text[0])
	return 0
}

func (r *MultiverseRenderer) multiverseBacktick(out *bytes.Buffer, previousChar byte, text []byte) int {
	if len(text) >= 2 && text[1] == '`' {
		nextChar := byte(0)
		if len(text) >= 3 {
			nextChar = text[2]
		}
		if multiverseQuoteHelper(out, previousChar, nextChar, 'd', &r.inDoubleQuote, false) {
			return 1
		}
	}

	out.WriteByte(text[0])
	return 0
}

func (r *MultiverseRenderer) multiverseNumberGeneric(out *bytes.Buffer, previousChar byte, text []byte) int {
	if wordBoundary(previousChar) && previousChar != '/' && len(text) >= 3 {
		// is it of the form digits/digits(word boundary)?, i.e., \d+/\d+\b
		// note: check for regular slash (/) or fraction slash (⁄, 0x2044, or 0xe2 81 84 in utf-8)
		//       and avoid changing dates like 1/23/2005 into fractions.
		numEnd := 0
		for len(text) > numEnd && isdigit(text[numEnd]) {
			numEnd++
		}
		if numEnd == 0 {
			out.WriteByte(text[0])
			return 0
		}
		denStart := numEnd + 1
		if len(text) > numEnd+3 && text[numEnd] == 0xe2 && text[numEnd+1] == 0x81 && text[numEnd+2] == 0x84 {
			denStart = numEnd + 3
		} else if len(text) < numEnd+2 || text[numEnd] != '/' {
			out.WriteByte(text[0])
			return 0
		}
		denEnd := denStart
		for len(text) > denEnd && isdigit(text[denEnd]) {
			denEnd++
		}
		if denEnd == denStart {
			out.WriteByte(text[0])
			return 0
		}
		if len(text) == denEnd || wordBoundary(text[denEnd]) && text[denEnd] != '/' {
			out.WriteString("<sup>")
			out.Write(text[:numEnd])
			out.WriteString("</sup>&frasl;<sub>")
			out.Write(text[denStart:denEnd])
			out.WriteString("</sub>")
			return denEnd - 1
		}
	}

	out.WriteByte(text[0])
	return 0
}

func (r *MultiverseRenderer) multiverseNumber(out *bytes.Buffer, previousChar byte, text []byte) int {
	if wordBoundary(previousChar) && previousChar != '/' && len(text) >= 3 {
		if text[0] == '1' && text[1] == '/' && text[2] == '2' {
			if len(text) < 4 || wordBoundary(text[3]) && text[3] != '/' {
				out.WriteString("&frac12;")
				return 2
			}
		}

		if text[0] == '1' && text[1] == '/' && text[2] == '4' {
			if len(text) < 4 || wordBoundary(text[3]) && text[3] != '/' || (len(text) >= 5 && tolower(text[3]) == 't' && tolower(text[4]) == 'h') {
				out.WriteString("&frac14;")
				return 2
			}
		}

		if text[0] == '3' && text[1] == '/' && text[2] == '4' {
			if len(text) < 4 || wordBoundary(text[3]) && text[3] != '/' || (len(text) >= 6 && tolower(text[3]) == 't' && tolower(text[4]) == 'h' && tolower(text[5]) == 's') {
				out.WriteString("&frac34;")
				return 2
			}
		}
	}

	out.WriteByte(text[0])
	return 0
}

func (r *MultiverseRenderer) multiverseDoubleQuoteVariant(out *bytes.Buffer, previousChar byte, text []byte, quote byte) int {
	nextChar := byte(0)
	if len(text) > 1 {
		nextChar = text[1]
	}
	if !multiverseQuoteHelper(out, previousChar, nextChar, quote, &r.inDoubleQuote, false) {
		out.WriteString("&quot;")
	}

	return 0
}

func (r *MultiverseRenderer) multiverseDoubleQuote(out *bytes.Buffer, previousChar byte, text []byte) int {
	return r.multiverseDoubleQuoteVariant(out, previousChar, text, 'd')
}

func (r *MultiverseRenderer) multiverseAngledDoubleQuote(out *bytes.Buffer, previousChar byte, text []byte) int {
	return r.multiverseDoubleQuoteVariant(out, previousChar, text, 'a')
}

func (r *MultiverseRenderer) multiverseLeftAngle(out *bytes.Buffer, previousChar byte, text []byte) int {
	i := 0

	for i < len(text) && text[i] != '>' {
		i++
	}

	out.Write(text[:i+1])
	return i
}

type multiverseCallback func(out *bytes.Buffer, previousChar byte, text []byte) int

// NewMultiverseRenderer constructs a Multiverse renderer object.
func NewMultiverseRenderer(flags HTMLFlags) *MultiverseRenderer {
	var (
		r MultiverseRenderer

		multiverseAmpAngled      = r.multiverseAmp(true, false)
		multiverseAmpAngledNBSP  = r.multiverseAmp(true, true)
		multiverseAmpRegular     = r.multiverseAmp(false, false)
		multiverseAmpRegularNBSP = r.multiverseAmp(false, true)

		addNBSP = flags&MultiverseQuotesNBSP != 0
	)

	if flags&MultiverseAngledQuotes == 0 {
		r.callbacks['"'] = r.multiverseDoubleQuote
		if !addNBSP {
			r.callbacks['&'] = multiverseAmpRegular
		} else {
			r.callbacks['&'] = multiverseAmpRegularNBSP
		}
	} else {
		r.callbacks['"'] = r.multiverseAngledDoubleQuote
		if !addNBSP {
			r.callbacks['&'] = multiverseAmpAngled
		} else {
			r.callbacks['&'] = multiverseAmpAngledNBSP
		}
	}
	r.callbacks['\''] = r.multiverseSingleQuote
	r.callbacks['('] = r.multiverseParens
	if flags&MultiverseDashes != 0 {
		if flags&MultiverseLatexDashes == 0 {
			r.callbacks['-'] = r.multiverseDash
		} else {
			r.callbacks['-'] = r.multiverseDashLatex
		}
	}
	r.callbacks['.'] = r.multiversePeriod
	if flags&MultiverseFractions == 0 {
		r.callbacks['1'] = r.multiverseNumber
		r.callbacks['3'] = r.multiverseNumber
	} else {
		for ch := '1'; ch <= '9'; ch++ {
			r.callbacks[ch] = r.multiverseNumberGeneric
		}
	}
	r.callbacks['<'] = r.multiverseLeftAngle
	r.callbacks['`'] = r.multiverseBacktick
	return &r
}

// Process is the entry point of the Multiverse renderer.
func (r *MultiverseRenderer) Process(w io.Writer, text []byte) {
	mark := 0
	for i := 0; i < len(text); i++ {
		if action := r.callbacks[text[i]]; action != nil {
			if i > mark {
				w.Write(text[mark:i])
			}
			previousChar := byte(0)
			if i > 0 {
				previousChar = text[i-1]
			}
			var tmp bytes.Buffer
			i += action(&tmp, previousChar, text[i:])
			w.Write(tmp.Bytes())
			mark = i + 1
		}
	}
	if mark < len(text) {
		w.Write(text[mark:])
	}
}
