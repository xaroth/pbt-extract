package dds_test

import (
	"encoding/hex"
	"image/color"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/xaroth/pbt-extract/dds"
	"github.com/xaroth/pbt-extract/dds/ddstest"
)

func TestDecodeBC7Block(t *testing.T) {
	t.Parallel()
	// Random blocks; the pixels are what Pillow 13 decodes them to.
	testCases := []struct {
		name  string
		block string
		want  [16][4]uint8
	}{
		{
			name:  "mode 1, two subsets, shared p-bits",
			block: "52f22665a60c12d289185d950ee88136",
			want: [16][4]uint8{
				{164, 175, 110, 255}, {136, 190, 135, 255}, {80, 97, 30, 255}, {80, 97, 30, 255},
				{190, 162, 86, 255}, {136, 190, 135, 255}, {164, 175, 110, 255}, {72, 129, 32, 255},
				{203, 155, 74, 255}, {136, 190, 135, 255}, {110, 203, 159, 255}, {203, 155, 74, 255},
				{203, 155, 74, 255}, {136, 190, 135, 255}, {136, 190, 135, 255}, {190, 162, 86, 255},
			},
		},
		{
			name:  "mode 4, rotation and index selection",
			block: "70166f6b113d178d6c0fd3901ff239a1",
			want: [16][4]uint8{
				{187, 209, 210, 143}, {192, 194, 210, 104}, {181, 222, 209, 181}, {192, 194, 211, 104},
				{192, 194, 210, 104}, {187, 209, 207, 143}, {181, 222, 207, 181}, {187, 209, 211, 143},
				{192, 194, 210, 104}, {187, 209, 208, 143}, {198, 181, 207, 66}, {192, 194, 209, 104},
				{198, 181, 209, 66}, {187, 209, 210, 143}, {181, 222, 211, 181}, {192, 194, 208, 104},
			},
		},
		{
			name:  "mode 6, 4-bit indices",
			block: "c095f20f9395650cf9380b8edb224a6b",
			want: [16][4]uint8{
				{103, 212, 127, 80}, {149, 97, 203, 25}, {119, 171, 155, 60}, {99, 222, 121, 85},
				{132, 139, 176, 45}, {86, 254, 100, 100}, {145, 107, 197, 30}, {119, 171, 155, 60},
				{132, 139, 176, 45}, {140, 119, 189, 36}, {95, 232, 114, 89}, {95, 232, 114, 89},
				{128, 149, 169, 50}, {103, 212, 127, 80}, {132, 139, 176, 45}, {112, 190, 142, 70},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			block, err := hex.DecodeString(testCase.block)
			if err != nil {
				t.Fatal(err)
			}
			img, err := dds.Decode(ddstest.BC7(4, 4, block))
			if err != nil {
				t.Fatal(err)
			}
			var got [16][4]uint8
			for i := range got {
				copy(got[i][:], img.Pix[img.PixOffset(i%4, i/4):])
			}
			if diff := cmp.Diff(testCase.want, got); diff != "" {
				t.Errorf("pixels mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDecodeCropsToSize(t *testing.T) {
	t.Parallel()
	red, blue := color.NRGBA{R: 254, A: 254}, color.NRGBA{B: 254, A: 254}
	img, err := dds.Decode(ddstest.BC7(6, 3, ddstest.Solid(red), ddstest.Solid(blue)))
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Size(); got.X != 6 || got.Y != 3 {
		t.Fatalf("size %v, want 6x3", got)
	}
	if got := img.NRGBAAt(3, 2); got != red {
		t.Errorf("(3,2) = %v, want %v", got, red)
	}
	if got := img.NRGBAAt(5, 2); got != blue {
		t.Errorf("(5,2) = %v, want %v", got, blue)
	}
}

func TestDecodeRejects(t *testing.T) {
	t.Parallel()
	notBC7 := ddstest.BC7(4, 4, make([]byte, 16))
	notBC7[128] = 71 // BC1
	for name, b := range map[string][]byte{
		"not DDS":   []byte("PNG\x00"),
		"not BC7":   notBC7,
		"cut short": ddstest.BC7(8, 4, make([]byte, 16)),
	} {
		if _, err := dds.Decode(b); err == nil {
			t.Errorf("%s: got nil error", name)
		}
	}
}
