package monitor

import (
	"context"
	"errors"
	"time"

	"github.com/glvr182/appie"
)

var (
	// ErrAlreadyWatching indicates that the product is already monitored
	ErrAlreadyWatching = errors.New("already watching product")
	// ErrNotWatching indicates that the product is not yet monitored
	ErrNotWatching = errors.New("not watching this product")
)

// Monitor contains the monitor state
type Monitor struct {
	watching []appie.Product
	out      chan appie.Product
	ctx      context.Context
	cancel   context.CancelFunc
}

// New returns a new monitor instance
func New(out chan appie.Product) (*Monitor, error) {
	mon := new(Monitor)
	mon.watching = make([]appie.Product, 0)
	mon.out = out
	mon.ctx, mon.cancel = context.WithCancel(context.Background())
	return mon, nil
}

// Run starts the monitor
func (m *Monitor) Run() error {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-m.ctx.Done():
			return nil
		case <-ticker.C:
			for i, target := range m.watching {
				product, err := appie.ProductByID(target.ID)
				if err != nil {
					return err
				}
				if (target.Price.Now != product.Price.Now || target.Control.Theme != product.Control.Theme) && product.Discount.Theme != "" {
					m.watching[i] = product
					m.out <- product
				}
			}
		}
	}
}

// Stop stops the monitor
func (m *Monitor) Stop() error {
	m.cancel()
	return nil
}

// Watch adds the product to the watch list
func (m *Monitor) Watch(pid int) error {
	_, found := contains(m.watching, pid)
	if found {
		return ErrAlreadyWatching
	}

	product, err := appie.ProductByID(pid)
	if err != nil {
		return err
	}
	m.watching = append(m.watching, product)

	return nil
}

// Unwatch removes the product from the watch list
func (m *Monitor) Unwatch(pid int) error {
	index, found := contains(m.watching, pid)
	if !found {
		return ErrNotWatching
	}

	m.watching = remove(m.watching, index)
	return nil
}

// contains checks if the product is being watched
func contains(s []appie.Product, e int) (int, bool) {
	for i, a := range s {
		if a.ID == e {
			return i, true
		}
	}
	return -1, false
}

// remove removes the product at index from the list
func remove(slice []appie.Product, i int) []appie.Product {
	copy(slice[i:], slice[i+1:])
	return slice[:len(slice)-1]
}
