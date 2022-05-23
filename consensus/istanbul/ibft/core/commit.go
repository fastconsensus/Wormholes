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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/istanbul"
	istanbulcommon "github.com/ethereum/go-ethereum/consensus/istanbul/common"
	ibfttypes "github.com/ethereum/go-ethereum/consensus/istanbul/ibft/types"
)

func (c *core) sendCommit() {
	sub := c.current.Subject()
	miniredis.GetLogCh() <- map[string]string{"sendCommit" + sub.View.Sequence.String():
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
	log.Info("carver|sendCommit|baseinfo",
		"no", sub.View.Sequence.Uint64(),
		"round", sub.View.Round.String(),
		"hash", sub.Digest.Hex(),
		"author", c.Address().Hex())
	c.broadcastCommit(sub)
}

func (c *core) sendCommitForOldBlock(view *istanbul.View, digest common.Hash) {
	sub := &istanbul.Subject{
		View:   view,
		Digest: digest,
	}
	c.broadcastCommit(sub)
}

func (c *core) broadcastCommit(sub *istanbul.Subject) {
	logger := c.logger.New("state", c.state)

	encodedSubject, err := ibfttypes.Encode(sub)
	if err != nil {
		logger.Error("Failed to encode", "subject", sub)
		return
	}
	c.broadcast(&ibfttypes.Message{
		Code: ibfttypes.MsgCommit,
		Msg:  encodedSubject,
	})
}

func (c *core) handleCommit(msg *ibfttypes.Message, src istanbul.Validator) error {
	// Decode COMMIT message
	var commit *istanbul.Subject
	err := msg.Decode(&commit)
	if err != nil {
		log.Error("caver|handleCommit|Decodecommit")
		return istanbulcommon.ErrFailedDecodeCommit
	}

	log.Info("carver|handleCommit|baseinfo",  "no", commit.View.Sequence,
		"round", commit.View.Round,
		"from", src.Address().Hex(),
		"hash", commit.Digest.Hex())

	if err := c.checkMessage(ibfttypes.MsgCommit, commit.View); err != nil {
		log.Error("caver|handleCommit|checkMessage", "no", commit.View.Sequence,
			"round", commit.View.Round,
			"who", c.address.Hex(),
			"hash", commit.Digest.Hex(),
			"err", err.Error())
		return err
	}

	if err := c.verifyCommit(commit, src); err != nil {
		log.Error("caver|handleCommit|verifyCommit", "no", commit.View.Sequence, "round", commit.View.Round, "who", c.address.Hex(),"hash", commit.Digest.Hex(), "err", err.Error())
		return err
	}

	miniredis.GetLogCh() <- map[string]string{"handleCommit" + commit.View.Sequence.String():
		" time|" + time.Now().Format("2006-01-02 03:04:05") +
		" no|" + commit.View.Sequence.String() +
		" round|" + commit.View.Round.String() +
		" src|" + src.Address().Hex() +
		" receiver|" + c.address.Hex() +
		" hash|" + commit.Digest.Hex() +
		" isProposer|" +
		strconv.FormatBool(c.IsProposer()),
	}

	c.acceptCommit(msg, src)
	log.Info("carver|handleCommit|baseinfo",  "no",  commit.View.Sequence.Uint64(), "round", commit.View.Round, "from", src.Address().Hex(),"hash", commit.Digest.Hex())
	// Commit the proposal once we have enough COMMIT messages and we are not in the Committed state.
	//
	// If we already have a proposal, we may have chance to speed up the consensus process
	// by committing the proposal without PREPARE messages.
	if c.current.Commits.Size() >= c.QuorumSize() && c.state.Cmp(ibfttypes.StateCommitted) < 0 {
		// Still need to call LockHash here since state can skip Prepared state and jump directly to the Committed state.
		log.Info("caver|handleCommit|commit",
			"no", commit.View.Sequence,
			"round", commit.View.Round,
			"CommitsSize", c.current.Commits.Size(),
			"hash", commit.Digest.Hex(),
		)
		c.current.LockHash()
		c.commit()
	}

	return nil
}

// verifyCommit verifies if the received COMMIT message is equivalent to our subject
func (c *core) verifyCommit(commit *istanbul.Subject, src istanbul.Validator) error {
	logger := c.logger.New("from", src, "state", c.state)

	sub := c.current.Subject()
	if !reflect.DeepEqual(commit, sub) {
		logger.Warn("Inconsistent subjects between commit and proposal", "expected", sub, "got", commit)
		return istanbulcommon.ErrInconsistentSubject
	}

	return nil
}

func (c *core) acceptCommit(msg *ibfttypes.Message, src istanbul.Validator) error {
	logger := c.logger.New("from", src, "state", c.state)

	// Add the COMMIT message to current round state
	if err := c.current.Commits.Add(msg); err != nil {
		logger.Error("Failed to record commit message", "msg", msg, "err", err)
		return err
	}

	return nil
}
