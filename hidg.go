package hidg

import (
	"os"
	"sync"
)

// A Device provides access to a HID gadget device.
type Device interface {
	// Close closes the device and associated resources.
	Close()

	// Write writes an output report to device. The first byte must be the
	// report number to write, zero if the device does not use numbered reports.
	Write([]byte) error

	// ReadCh returns a channel that will be sent input reports from the device.
	// If the device uses numbered reports, the first byte will be the report
	// number.
	ReadCh() <-chan []byte

	// ReadError returns the read error, if any after the channel returned from
	// ReadCh has been closed.
	ReadError() error
}

type hidgDevice struct {
	f *os.File

	readSetup sync.Once
	readErr   error
	readCh    chan []byte
}

func Open(path string) (Device, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	d := &hidgDevice{
		f: f,
	}
	return d, nil
}

func (d *hidgDevice) Close() {
	d.f.Close()
}

func (d *hidgDevice) Write(data []byte) error {
	_, err := d.f.Write(data)
	return err
}

func (d *hidgDevice) ReadCh() <-chan []byte {
	d.readSetup.Do(func() {
		d.readCh = make(chan []byte, 64)
		go d.readThread()
	})
	return d.readCh
}

func (d *hidgDevice) ReadError() error {
	return d.readErr
}

func (d *hidgDevice) readThread() {
	defer close(d.readCh)
	for {
		buf := make([]byte, 64)
		n, err := d.f.Read(buf)
		if err != nil {
			d.readErr = err
			return
		}
		select {
		case d.readCh <- buf[:n]:
		default:
		}
	}
}
