package store

import (
	"context"
	"testing"
)

func TestCreationMaterialConstraintsDoNotRequireAQuote(t *testing.T) {
	image := SourceRefItem{Type: "pending", ID: "local-image"}
	for _, scenario := range []struct {
		name        string
		constraints creationSourceConstraints
		meta        SourceMeta
		want        string
	}{
		{"text", creationSourceConstraints{ModeCode: "text_to_video"}, SourceMeta{}, ""},
		{"text_shape", creationSourceConstraints{ModeCode: "text_to_video"}, SourceMeta{Refs: []SourceRefItem{image}}, "文生视频不支持图片素材"},
		{"minimum", creationSourceConstraints{ModeCode: "multi_image_to_video", MinImages: 2, MaxImages: 4}, SourceMeta{Refs: []SourceRefItem{image}}, "多图生成视频至少需要 2 张参考图"},
		{"maximum", creationSourceConstraints{ModeCode: "multi_image_to_video", MinImages: 1, MaxImages: 1}, SourceMeta{Refs: []SourceRefItem{image, image}}, "参考图最多 1 张"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// No quote ID, price, expiry, or database is needed to reject an
			// invalid material shape or enforce execution image limits.
			err := validateModeInput(context.Background(), nil, 1, scenario.constraints, "镜头描述", &scenario.meta)
			if scenario.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || err.Error() != scenario.want {
				t.Fatalf("want %q, got %v", scenario.want, err)
			}
		})
	}
}

func TestCreationSourceShapeAndSlotCount(t *testing.T) {
	image := SourceRefItem{Type: "pending", ID: "local-image"}
	audio := SourceRefItem{Type: "upload", ID: "audio-only"}
	for _, scenario := range []struct {
		name  string
		mode  string
		meta  SourceMeta
		count int
		valid bool
	}{
		{"text_without_images", "text_to_video", SourceMeta{}, 0, true},
		{"audio_is_not_an_image", "text_to_video", SourceMeta{Audio: &audio}, 0, true},
		{"text_rejects_references", "text_to_video", SourceMeta{Refs: []SourceRefItem{image}}, 1, false},
		{"first_last_same_image_occupies_both_slots", "first_last_frame_to_video", SourceMeta{First: &image, Last: &image}, 2, true},
		{"first_last_requires_end", "first_last_frame_to_video", SourceMeta{First: &image}, 1, false},
		{"first_last_rejects_two_refs_in_wrong_slots", "first_last_frame_to_video", SourceMeta{Refs: []SourceRefItem{image, image}}, 2, false},
		{"refs_do_not_hide_first_last_slots", "first_last_frame_to_video", SourceMeta{First: &image, Last: &image, Refs: []SourceRefItem{image}}, 3, false},
		{"first_last_rejects_extra_slot", "first_last_frame_to_video", SourceMeta{First: &image, Last: &image, Extras: []SourceRefItem{image}}, 3, false},
		{"multi_pending_reference_allowed", "multi_image_to_video", SourceMeta{Refs: []SourceRefItem{image}}, 1, true},
		{"multi_requires_reference", "multi_image_to_video", SourceMeta{}, 0, false},
		{"multi_rejects_first_frame_slot", "multi_image_to_video", SourceMeta{First: &image}, 1, false},
		{"multi_rejects_mixed_shape", "multi_image_to_video", SourceMeta{First: &image, Refs: []SourceRefItem{image}}, 2, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if count := scenario.meta.ImageCount(); count != scenario.count {
				t.Fatalf("image slots: got=%d want=%d", count, scenario.count)
			}
			if err := ValidateCreationSourceShape(scenario.mode, scenario.meta); (err == nil) != scenario.valid {
				t.Fatalf("valid=%v err=%v", scenario.valid, err)
			}
		})
	}
}
