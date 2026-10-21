package service

import (
	"testing"

	"github.com/Maluslock/comic/server/internal/repository"
)

func TestBookingToItem_MapsServiceDurationSnapshot(t *testing.T) {
	d := int32(120)
	item := bookingToItem(repository.Booking{ID: 1, ServiceDuration: &d})
	if item.ServiceDuration == nil {
		t.Fatal("serviceDuration must be carried from the booking snapshot; the order page shows 0 分钟 without it")
	}
	if *item.ServiceDuration != 120 {
		t.Errorf("want 120, got %d", *item.ServiceDuration)
	}
}

func TestBookingToItem_KeepsDurationNilWhenSnapshotMissing(t *testing.T) {
	item := bookingToItem(repository.Booking{ID: 1})
	if item.ServiceDuration != nil {
		t.Errorf("want nil when the snapshot is absent, got %v", *item.ServiceDuration)
	}
}
