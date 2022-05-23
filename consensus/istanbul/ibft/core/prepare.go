// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miniredis"
	"reflect"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/consensus/istanbul"
	istanbulcommon "github.com/ethereum/go-ethereum/consensus/istanbul/common"
	ibfttypes "github.com/ethereum/go-ethereum/consensus/istanbul/ibft/types"
)

func (c *core) sendPrepare() {
	logger := c.logger.New("state", c.state)

	sub := c.current.Subject()
	encodedSubject, err := ibfttypes.Encode(sub)
	if err != nil {
		logger.Error("Failed to encode", "subject", sub)
		return
	}

	miniredis.GetLogCh() <- map[string]string{"sendPrepare" + sub.View.Sequence.String():
		" time|" +
		time.Now().Format("2006-01-02 03:04:05") +
		" no|" +
		sub.View.Sequence.String() +
		" round|" +
		sub.View.Round.String() +
		" author|" +
		c.address.Hex() +
		" hash|" +
		sub.Digest.Hex() +
		" isProposer|" +
		strconv.FormatBool(c.IsProposer()),
	}
	log.Info("carver|sendPrepare|baseinfo",
		"no", sub.View.Sequence.Uint64(),
		"round", sub.View.Round.String(),
		"hash", sub.Digest.Hex(),
		"author", c.Address().Hex())
	c.broadcast(&ibfttypes.Message{
		Code: ibfttypes.MsgPrepare,
		Msg:  encodedSubject,
	})
}

func (c *core) handlePrepare(msg *ibfttypes.Message, src istanbul.Validator) error {
	// Decode PREPARE message
	var prepare *istanbul.Subject
	err := msg.Decode(&prepare)
	if err != nil {
		return istanbulcommon.ErrFailedDecodePrepare
	}

	log.Info("carver|handlePrepare|baseinfo",  "no", prepare.View.Sequence,
		"round", prepare.View.Round,
		"from", src.Address().Hex(),
		"hash", prepare.Digest.Hex())

	if err := c.checkMessage(ibfttypes.MsgPrepare, prepare.View); err != nil {
		log.Error("caver|handlePrepare|checkMessage", "no", prepare.View.Sequence,
			"round", prepare.View.Round,
			"from", src.Address().Hex(),
			"hash", prepare.Digest.Hex(),
			"err", err.Error())
		return err
	}

	// If it is locked, it can only process on the locked block.
	// Passing verifyPrepare and checkMessage implies it is processing on the locked block since it was verified in the Preprepared state.
	if err := c.verifyPrepare(prepare, src); err != nil {
		log.Info("carver|handlePrepare|verifyPrepare", "no", prepare.View.Sequence,
			"round", prepare.View.Round.String(),
			"hash", prepare.Digest.Hex(),
			"err", err)
		return err
	}

	miniredis.GetLogCh() <- map[string]string{"handlePrepare" + prepare.View.Sequence.String():
		" time|" + time.Now().Format("2006-01-02 03:04:05") +
		" no|" + prepare.View.Sequence.String() +
		" round|" + prepare.View.Round.String() +
		" src|" + src.Address().Hex() +
		" receiver|" + c.address.Hex() +
		" hash|" + prepare.Digest.Hex() +
		" isProposer|" +
		strconv.FormatBool(c.IsProposer()),
	}

	c.acceptPrepare(msg, src)
	// Change to Prepared state if we've received enough PREPARE messages or it is locked
	// and we are in earlier state before Prepared state.
	if ((c.current.IsHashLocked() && prepare.Digest == c.current.GetLockedHash()) || c.current.GetPrepareOrCommitSize() >= c.QuorumSize()) &&
		c.state.Cmp(ibfttypes.StatePrepared) < 0 {
		c.current.LockHash()
		c.setState(ibfttypes.StatePrepared)
		log.Info("caver|handlePrepare|sendCommit",
			"no", prepare.View.Sequence,
			"round", prepare.View.Round,
			"prepare+commitSize", c.current.GetPrepareOrCommitSize(),
			"hash", prepare.Digest.Hex(),
			"self", c.address.Hex(),
		)
		c.sendCommit()
	}

	return nil
}

// verifyPrepare verifies if the received PREPARE message is equivalent to our subject
func (c *core) verifyPrepare(prepare *istanbul.Subject, src istanbul.Validator) error {
	logger := c.logger.New("from", src, "state", c.state)

	sub := c.current.Subject()
	if !reflect.DeepEqual(prepare, sub) {
		logger.Warn("Inconsistent subjects between PREPARE and proposal", "expected", sub, "got", prepare)
		return istanbulcommon.ErrInconsistentSubject
	}

	return nil
}

func (c *core) acceptPrepare(msg *ibfttypes.Message, src istanbul.Validator) error {
	logger := c.logger.New("from", src, "state", c.state)

	// Add the PREPARE message to current round state
	if err := c.current.Prepares.Add(msg); err != nil {
		logger.Error("Failed to add PREPARE message to round state", "msg", msg, "err", err)
		return err
	}

	return nil
}