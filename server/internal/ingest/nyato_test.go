package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func loadFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "sample.html"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(b)
}

func TestStripThumbSuffix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg!330x450cut",
			want:  "https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg",
		},
		{
			input: "https://img.nyato.com/data/upload/expo/2026/0601/15/abc123def456.jpg",
			want:  "https://img.nyato.com/data/upload/expo/2026/0601/15/abc123def456.jpg",
		},
		{
			input: "https://img.nyato.com/other.png!330x450cut",
			want:  "https://img.nyato.com/other.png",
		},
	}
	for _, tt := range tests {
		got := stripThumbSuffix(tt.input)
		if got != tt.want {
			t.Errorf("stripThumbSuffix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseEventCards(t *testing.T) {
	html := loadFixture(t)
	cards, err := ParseEventCards(html)
	if err != nil {
		t.Fatalf("ParseEventCards: %v", err)
	}
	if len(cards) != 3 {
		t.Fatalf("expected 3 cards, got %d", len(cards))
	}

	// Card 1: 西安星幻动漫节
	c1 := cards[0]
	if c1.Name != "2026第19届西安星幻动漫节" {
		t.Errorf("card 1 Name = %q, want %q", c1.Name, "2026第19届西安星幻动漫节")
	}
	if c1.City != "西安市" {
		t.Errorf("card 1 City = %q, want %q", c1.City, "西安市")
	}
	if c1.DateStart != "02/21" {
		t.Errorf("card 1 DateStart = %q, want %q", c1.DateStart, "02/21")
	}
	if c1.DateEnd != "02/21" {
		t.Errorf("card 1 DateEnd = %q, want %q", c1.DateEnd, "02/21")
	}
	if c1.Address != "陕西省 西安市 雁塔区 西安国际会展中心" {
		t.Errorf("card 1 Address = %q, want %q", c1.Address, "陕西省 西安市 雁塔区 西安国际会展中心")
	}
	if c1.ImageURL != "https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg" {
		t.Errorf("card 1 ImageURL = %q, want stripped URL", c1.ImageURL)
	}
	if c1.Venue != "" {
		t.Errorf("card 1 Venue = %q, want empty (no venue source)", c1.Venue)
	}

	// Card 2: CP30魔都同人祭
	c2 := cards[1]
	if c2.Name != "CP30魔都同人祭" {
		t.Errorf("card 2 Name = %q, want %q", c2.Name, "CP30魔都同人祭")
	}
	if c2.City != "上海市" {
		t.Errorf("card 2 City = %q, want %q", c2.City, "上海市")
	}
	if c2.DateStart != "05/01" {
		t.Errorf("card 2 DateStart = %q, want %q", c2.DateStart, "05/01")
	}
	if c2.DateEnd != "05/03" {
		t.Errorf("card 2 DateEnd = %q, want %q", c2.DateEnd, "05/03")
	}
	if c2.ImageURL != "https://img.nyato.com/data/upload/expo/2026/0601/15/abc123def456.jpg" {
		t.Errorf("card 2 ImageURL = %q, want stripped URL", c2.ImageURL)
	}

	// Card 3: 成都COMIDAY26
	c3 := cards[2]
	if c3.Name != "2026成都COMIDAY26动漫展" {
		t.Errorf("card 3 Name = %q, want %q", c3.Name, "2026成都COMIDAY26动漫展")
	}
	if c3.City != "成都市" {
		t.Errorf("card 3 City = %q, want %q", c3.City, "成都市")
	}
	if c3.DateStart != "07/12" {
		t.Errorf("card 3 DateStart = %q, want %q", c3.DateStart, "07/12")
	}
	if c3.DateEnd != "07/13" {
		t.Errorf("card 3 DateEnd = %q, want %q", c3.DateEnd, "07/13")
	}
	if c3.ImageURL != "https://img.nyato.com/data/upload/expo/2026/0615/08/xyz789ghi012.jpg" {
		t.Errorf("card 3 ImageURL = %q, want stripped URL", c3.ImageURL)
	}
}

// TestParseEventCardsFiltersDemoCard asserts the nyato demo card
// ("喵特门票购买演示", which has empty city/dates) is excluded from output.
func TestParseEventCardsFiltersDemoCard(t *testing.T) {
	cards, err := ParseEventCards(loadFixture(t))
	if err != nil {
		t.Fatalf("ParseEventCards: %v", err)
	}
	for _, c := range cards {
		if c.Name == "喵特门票购买演示" {
			t.Errorf("demo card %q should be filtered out (empty city/dates)", c.Name)
		}
		if c.Name != "" && (c.City == "" || c.DateStart == "" || c.DateEnd == "") {
			t.Errorf("card %q kept but missing required field: city=%q start=%q end=%q",
				c.Name, c.City, c.DateStart, c.DateEnd)
		}
	}
}
