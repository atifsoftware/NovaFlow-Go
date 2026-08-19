package core

import (
	"math"
	"strings"
)

// NumberToWordsOptions configures the language and currency representation.
type NumberToWordsOptions struct {
	Language string // "en" (default) or "bn"
	Currency string // "BDT", "USD", "EUR", or custom name
}

var enUnits = []string{
	"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine",
	"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen",
}

var enTens = []string{
	"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety",
}

var enScales = []string{"", "Thousand", "Million", "Billion", "Trillion"}

var bnNumbers = []string{
	"শূন্য", "এক", "দুই", "তিন", "চার", "পাঁচ", "ছয়", "সাত", "আট", "নয়",
	"দশ", "এগার", "বার", "তের", "চৌদ্দ", "পনের", "ষোল", "সতের", "আঠার", "উনিশ",
	"বিশ", "একুশ", "বাইশ", "তেইশ", "চব্বিশ", "পঁচিশ", "ছাব্বিশ", "সাতাশ", "আটাশ", "উনত্রিশ",
	"ত্রিশ", "একত্রিশ", "বত্রিশ", "তেত্রিশ", "চৌত্রিশ", "পঁয়ত্রিশ", "ছত্রিশ", "সাঁইত্রিশ", "আটত্রিশ", "উনচল্লিশ",
	"চল্লিশ", "একচল্লিশ", "বিয়াল্লিশ", "তেতাল্লিশ", "চুয়াল্লিশ", "পঁয়তাল্লিশ", "ছেচল্লিশ", "সাতচল্লিশ", "আটচল্লিশ", "উনপঞ্চাশ",
	"পঞ্চাশ", "একান্ন", "বায়ান্ন", "তিপ্পান্ন", "চুয়ান্ন", "পঞ্চান্ন", "ছাপ্পান্ন", "সাতান্ন", "আটান্ন", "উনষাট",
	"ষাট", "একষট্টি", "বাষট্টি", "তেষট্টি", "চৌষট্টি", "পঁয়ষট্টি", "ছেষট্টি", "সাতষট্টি", "আটষট্টি", "উনসত্তর",
	"সত্তর", "একাত্তর", "বাহাত্তর", "তিয়াত্তর", "চুয়াত্তর", "পঁচাত্তর", "ছিয়াত্তর", "সাতাত্তর", "আটাত্তর", "উনআশি",
	"আশি", "একাশি", "বিরাশি", "তিরাশি", "চুরাশি", "পঁচাশি", "ছিয়াশি", "সাতাশি", "অষ্টআশি", "উননব্বই",
	"নব্বই", "একানব্বই", "বানব্বই", "তিরানব্বই", "চুরানব্বই", "পঁচানব্বই", "ছিয়ানব্বই", "সাতানব্বই", "আটানব্বই", "নিরানব্বই",
}

// NumberToWords converts a numerical amount into words in English or Bengali.
func NumberToWords(amount float64, opts ...NumberToWordsOptions) string {
	opt := NumberToWordsOptions{Language: "en", Currency: "BDT"}
	if len(opts) > 0 {
		if opts[0].Language != "" {
			opt.Language = strings.ToLower(opts[0].Language)
		}
		if opts[0].Currency != "" {
			opt.Currency = strings.ToUpper(opts[0].Currency)
		}
	}

	if opt.Language == "bn" || opt.Language == "bangla" || opt.Language == "bengali" {
		return formatBengaliAmount(amount, opt.Currency)
	}

	return formatEnglishAmount(amount, opt.Currency)
}

// NumberToWordsEn converts a number to English words with Taka/Paisa currency.
func NumberToWordsEn(amount float64) string {
	return NumberToWords(amount, NumberToWordsOptions{Language: "en", Currency: "BDT"})
}

// NumberToWordsBn converts a number to Bengali words with টাকা/পয়সা currency.
func NumberToWordsBn(amount float64) string {
	return NumberToWords(amount, NumberToWordsOptions{Language: "bn", Currency: "BDT"})
}

// --- English Implementation ---

func formatEnglishAmount(amount float64, currency string) string {
	if amount == 0 {
		return "Zero Taka Only"
	}

	isNegative := amount < 0
	amount = math.Abs(amount)

	intPart := int64(amount)
	decPart := int64(math.Round((amount - float64(intPart)) * 100))
	if decPart >= 100 {
		intPart++
		decPart -= 100
	}

	mainUnit := "Taka"
	subUnit := "Paisa"
	switch currency {
	case "USD":
		mainUnit = "Dollar"
		if intPart > 1 {
			mainUnit = "Dollars"
		}
		subUnit = "Cent"
		if decPart > 1 {
			subUnit = "Cents"
		}
	case "EUR":
		mainUnit = "Euro"
		if intPart > 1 {
			mainUnit = "Euros"
		}
		subUnit = "Cent"
		if decPart > 1 {
			subUnit = "Cents"
		}
	case "GBP":
		mainUnit = "Pound"
		if intPart > 1 {
			mainUnit = "Pounds"
		}
		subUnit = "Pence"
	}

	var sb strings.Builder
	if isNegative {
		sb.WriteString("Minus ")
	}

	if intPart > 0 {
		sb.WriteString(convertIntegerToEnWords(intPart))
		sb.WriteString(" " + mainUnit)
	}

	if decPart > 0 {
		if intPart > 0 {
			sb.WriteString(" and ")
		}
		sb.WriteString(convertIntegerToEnWords(decPart))
		sb.WriteString(" " + subUnit)
	}

	sb.WriteString(" Only")
	return strings.TrimSpace(sb.String())
}

func convertIntegerToEnWords(n int64) string {
	if n == 0 {
		return "Zero"
	}

	var parts []string
	scaleIndex := 0

	for n > 0 {
		chunk := n % 1000
		if chunk != 0 {
			chunkStr := convertThreeDigitsEn(chunk)
			if scaleIndex > 0 {
				chunkStr += " " + enScales[scaleIndex]
			}
			parts = append([]string{chunkStr}, parts...)
		}
		n /= 1000
		scaleIndex++
	}

	return strings.Join(parts, " ")
}

func convertThreeDigitsEn(n int64) string {
	var words []string

	hundreds := n / 100
	remainder := n % 100

	if hundreds > 0 {
		words = append(words, enUnits[hundreds], "Hundred")
	}

	if remainder > 0 {
		if remainder < 20 {
			words = append(words, enUnits[remainder])
		} else {
			tens := remainder / 10
			units := remainder % 10
			tenStr := enTens[tens]
			if units > 0 {
				tenStr += " " + enUnits[units]
			}
			words = append(words, tenStr)
		}
	}

	return strings.Join(words, " ")
}

// --- Bengali Implementation ---

func formatBengaliAmount(amount float64, currency string) string {
	if amount == 0 {
		return "শূন্য টাকা মাত্র"
	}

	isNegative := amount < 0
	amount = math.Abs(amount)

	intPart := int64(amount)
	decPart := int64(math.Round((amount - float64(intPart)) * 100))
	if decPart >= 100 {
		intPart++
		decPart -= 100
	}

	mainUnit := "টাকা"
	subUnit := "পয়সা"

	var sb strings.Builder
	if isNegative {
		sb.WriteString("মাইনাস ")
	}

	if intPart > 0 {
		sb.WriteString(convertIntegerToBnWords(intPart))
		sb.WriteString(" " + mainUnit)
	}

	if decPart > 0 {
		if intPart > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(convertIntegerToBnWords(decPart))
		sb.WriteString(" " + subUnit)
	}

	sb.WriteString(" মাত্র")
	return strings.TrimSpace(sb.String())
}

func convertIntegerToBnWords(n int64) string {
	if n == 0 {
		return "শূন্য"
	}

	var parts []string

	crore := n / 10000000
	n %= 10000000

	lakh := n / 100000
	n %= 100000

	thousand := n / 1000
	n %= 1000

	hundred := n / 100
	remainder := n % 100

	if crore > 0 {
		parts = append(parts, convertIntegerToBnWords(crore)+" কোটি")
	}
	if lakh > 0 {
		parts = append(parts, bnNumbers[lakh]+" লাখ")
	}
	if thousand > 0 {
		parts = append(parts, bnNumbers[thousand]+" হাজার")
	}
	if hundred > 0 {
		parts = append(parts, bnNumbers[hundred]+" শত")
	}
	if remainder > 0 {
		parts = append(parts, bnNumbers[remainder])
	}

	return strings.Join(parts, " ")
}
