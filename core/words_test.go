package core

import (
	"testing"
)

func TestNumberToWordsEn(t *testing.T) {
	tests := []struct {
		amount   float64
		expected string
	}{
		{0, "Zero Taka Only"},
		{5, "Five Taka Only"},
		{15, "Fifteen Taka Only"},
		{45, "Forty Five Taka Only"},
		{105, "One Hundred Five Taka Only"},
		{1234, "One Thousand Two Hundred Thirty Four Taka Only"},
		{15450.50, "Fifteen Thousand Four Hundred Fifty Taka and Fifty Paisa Only"},
		{1000000, "One Million Taka Only"},
		{2500000.75, "Two Million Five Hundred Thousand Taka and Seventy Five Paisa Only"},
		{-50, "Minus Fifty Taka Only"},
	}

	for _, tt := range tests {
		got := NumberToWordsEn(tt.amount)
		if got != tt.expected {
			t.Errorf("NumberToWordsEn(%f) = %q, expected %q", tt.amount, got, tt.expected)
		}
	}
}

func TestNumberToWordsEnCurrencies(t *testing.T) {
	usd := NumberToWords(100.50, NumberToWordsOptions{Language: "en", Currency: "USD"})
	if usd != "One Hundred Dollars and Fifty Cents Only" {
		t.Errorf("unexpected USD output: %q", usd)
	}

	eur := NumberToWords(1.01, NumberToWordsOptions{Language: "en", Currency: "EUR"})
	if eur != "One Euro and One Cent Only" {
		t.Errorf("unexpected EUR output: %q", eur)
	}
}

func TestNumberToWordsBn(t *testing.T) {
	tests := []struct {
		amount   float64
		expected string
	}{
		{0, "শূন্য টাকা মাত্র"},
		{5, "পাঁচ টাকা মাত্র"},
		{15, "পনের টাকা মাত্র"},
		{50, "পঞ্চাশ টাকা মাত্র"},
		{105, "এক শত পাঁচ টাকা মাত্র"},
		{1234, "এক হাজার দুই শত চৌত্রিশ টাকা মাত্র"},
		{15450.50, "পনের হাজার চার শত পঞ্চাশ টাকা পঞ্চাশ পয়সা মাত্র"},
		{100000, "এক লাখ টাকা মাত্র"},
		{15000000, "এক কোটি পঞ্চাশ লাখ টাকা মাত্র"},
		{-500, "মাইনাস পাঁচ শত টাকা মাত্র"},
	}

	for _, tt := range tests {
		got := NumberToWordsBn(tt.amount)
		if got != tt.expected {
			t.Errorf("NumberToWordsBn(%f) = %q, expected %q", tt.amount, got, tt.expected)
		}
	}
}
