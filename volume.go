package london

import (
	"context"
	"encoding/binary"
	"fmt" "time"

	"github.com/byuoitav/connpool"
)

const (
	volumeScaleFactor = 65536
)

func (d *DSP) GetVolumeByBlock(ctx context.Context, block string) (int, error) {
	subscribe, err := buildSubscribeCommand(methodSubscribePercent, stateGain, block, minSubscribeInterval)
	if err != nil {
		return 0, fmt.Errorf("unable to build subscribe command: %w", err)
	}

	unsubscribe, err := buildUnsubscribeCommand(methodUnsubscribePercent, stateGain, block)
	if err != nil {
		return 0, fmt.Errorf("unable to build unsubscribe command: %w", err)
	}

	var resp []byte

	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		n, err := conn.Write(subscribe)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write subscribe command: %w", err)
		case n != len(subscribe):
			return fmt.Errorf("unable to write subscribe command: wrote %v/%v bytes", n, len(subscribe))
		}

		resp, err = conn.ReadUntil(asciiETX, 3*time.Second)
		if err != nil {
			return fmt.Errorf("unable to read response: %w", err)
		}

		n, err = conn.Write(unsubscribe)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write unsubscribe command: %w", err)
		case n != len(unsubscribe):
			return fmt.Errorf("unable to write unsubscribe command: wrote %v/%v bytes", n, len(subscribe))
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	resp, err = decode(resp)
	if err != nil {
		return 0, fmt.Errorf("unable to decode response: %w", err)
	}

	data := resp[len(resp)-4:]
	vol := binary.BigEndian.Uint32(data)

	vol = vol / volumeScaleFactor
	vol++

	return int(vol), nil
}

func (d *DSP) SetVolumeByBlock(ctx context.Context, block string, volume int) error {
	volume *= volumeScaleFactor
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(volume))

	cmd, err := buildCommand(methodSetPercent, stateGain, block, data)
	if err != nil {
		return fmt.Errorf("unable to build command: %w", err)
	}

	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		n, err := conn.Write(cmd)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command: %w", err)
		case n != len(cmd):
			return fmt.Errorf("unable to write command: wrote %v/%v bytes", n, len(cmd))
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
