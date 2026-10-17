package lang

import (
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

// TestIndexErrorNamesStayDistinct pins the two runtime reason prefixes §8 defines: a container
// subscript is IndexOutOfBoundsError, a bit subscript and a reference moved past the end are
// IndexOutOfRangeError. Collapsing them would change a user-visible message, so each case also
// asserts the other name is absent.
func TestIndexErrorNamesStayDistinct(t *testing.T) {
	withLocalizer(t, i18n.New(i18n.EN))
	cases := []struct {
		name   string
		src    string
		want   string
		absent string
	}{
		{
			name:   "container subscript",
			src:    `fn main(IOStream io) { List<int> l = [1, 2]; int i = 5; io.println(l[i]); }`,
			want:   "IndexOutOfBoundsError: index 5 out of range [0,2)",
			absent: "IndexOutOfRangeError",
		},
		{
			name:   "bit subscript",
			src:    `fn main(IOStream io) { bits<8> b = 1; int i = 9; io.println(b[i]); }`,
			want:   "IndexOutOfRangeError: bit index 9 is out of range for bits<8> (0..7)",
			absent: "IndexOutOfBoundsError",
		},
		{
			name:   "reference move past the end",
			src:    `fn main(IOStream io) { List<int> l = [1, 2]; int& p = &l[0]; p++; p++; p++; }`,
			want:   "IndexOutOfRangeError: reference move past the end",
			absent: "IndexOutOfBoundsError",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runSrc(t, tc.src)
			if err == nil {
				t.Fatalf("expected %s, got no error", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.want)
			}
			if strings.Contains(err.Error(), tc.absent) {
				t.Fatalf("error %q also contains %q, so the names are not distinct", err.Error(), tc.absent)
			}
		})
	}
}
