// SPDX-License-Identifier: MPL-2.0
/*
 * Copyright (C) 2024 Damian Peckett <damian@pecke.tt>.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 *
 * Portions of this file are based on code originally from: github.com/paultag/go-debian
 *
 * Copyright (c) Paul R. Tagliamonte <paultag@debian.org>, 2015
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package deb822

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/ProtonMail/go-crypto/openpgp"
)

func Unmarshal(data []byte, v any) error {
	decoder, err := NewDecoder(bytes.NewReader(data), openpgp.EntityList{})
	if err != nil {
		return err
	}

	return decoder.Decode(v)
}

type Decoder struct {
	stanzaReader StanzaReader
}

func NewDecoder(reader io.Reader, keyring openpgp.EntityList) (*Decoder, error) {
	var ret Decoder
	pr, err := NewStanzaReader(reader, keyring)
	if err != nil {
		return nil, err
	}
	ret.stanzaReader = *pr
	return &ret, nil
}

// Return the Entity (if one exists) that signed this set of stanzas.
func (d *Decoder) Signer() *openpgp.Entity {
	return d.stanzaReader.Signer()
}

func (d *Decoder) Decode(v any) error {
	into := reflect.ValueOf(v)

	if into.Type().Kind() != reflect.Ptr {
		return errors.New("can't decode into a non-pointer")
	}

	switch into.Elem().Type().Kind() {
	case reflect.Struct:
		paragraph, err := d.stanzaReader.Next()
		if err != nil {
			return err
		}
		return decodeStruct(*paragraph, into)
	case reflect.Slice:
		return d.decodeSlice(into)
	default:
		return fmt.Errorf("can't decode into a %s", into.Elem().Type().Name())
	}
}

func (d *Decoder) decodeSlice(into reflect.Value) error {
	flavor := into.Elem().Type().Elem()

	for {
		targetValue := reflect.New(flavor)

		// Get the next stanza.
		stanza, err := d.stanzaReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if err := decodeStruct(*stanza, targetValue); err != nil {
			return err
		}
		into.Elem().Set(reflect.Append(into.Elem(), targetValue.Elem()))
	}
	return nil
}

func decodeStruct(stanza Stanza, into reflect.Value) error {
	// If we have a pointer, let's follow it.
	if into.Type().Kind() == reflect.Ptr {
		return decodeStruct(stanza, into.Elem())
	}

	// Marshal the stanza.
	jsonData, err := json.Marshal(stanza)
	if err != nil {
		return err
	}

	// Unmarshal the JSON into the struct.
	if err := json.Unmarshal(jsonData, into.Addr().Interface()); err != nil {
		return err
	}

	return nil
}
