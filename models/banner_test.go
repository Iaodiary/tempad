package models

import (
	"strconv"
	"testing"
)

func TestGetBannerByID(t *testing.T) {
	id := 1
	banner, err := GetBannerByID(id)
	if err != nil {
		t.Error(err)
	}
	if banner.ID == 0 {
		t.Log("Nonexsitence Banner" + strconv.Itoa(id))
	}
}

func TestGetBanners(t *testing.T) {
	size := "40*50"
	groupId := 1
	banners, err := GetBanners(size, groupId)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if len(banners) == 0 {
		t.Error("No match banner, not completely test")
	}
}
