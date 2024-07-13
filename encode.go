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
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

// Marshal is a one-off interface to serialize a single object to a writer.
//
// Most notably, this will *not* separate stanzas with a newline as is
// expected upon repeated calls, please use the Encoder streaming interface
// for that.
//
// Given a struct (or list of structs), write to the io.Writer stream
// in the RFC822-alike Debian control-file format
func Marshal(writer io.Writer, data any) error {
	encoder := NewEncoder(writer)
	return encoder.Encode(data)
}

// Encoder is a struct that allows for the streaming Encoding of data
// back out to an `io.Writer`. Most notably, this will separate
// subsequent `Encode` calls of a Struct with a newline.
//
// Given a struct (or list of structs), write to the io.Writer stream
// in the RFC822-alike Debian control-file format
//
// In order to Marshal a custom Struct, you are required to implement the
// Marshallable interface. It's highly encouraged to put this interface on
// the struct without a pointer receiver, so that pass-by-value works
// when you call Marshal.
type Encoder struct {
	writer         io.Writer
	alreadyWritten bool
}

// Create a new Encoder, which is configured to write to the given `io.Writer`.
func NewEncoder(writer io.Writer) *Encoder {
	return &Encoder{
		writer:         writer,
		alreadyWritten: false,
	}
}

// Take a Struct, Encode it into a stanza, and write that out to the
// io.Writer set up when the Encoder was configured.
func (e *Encoder) Encode(incoming interface{}) error {
	data := reflect.ValueOf(incoming)
	return e.encode(data)
}

func (e *Encoder) encode(data reflect.Value) error {
	if data.Type().Kind() == reflect.Ptr {
		return e.encode(data.Elem())
	}

	switch data.Type().Kind() {
	case reflect.Slice:
		return e.encodeSlice(data)
	case reflect.Struct:
		return e.encodeStruct(data)
	}
	return errors.New("unknown type")
}

func (e *Encoder) encodeSlice(data reflect.Value) error {
	for i := 0; i < data.Len(); i++ {
		if err := e.encodeStruct(data.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeStruct(data reflect.Value) error {
	if e.alreadyWritten {
		_, err := e.writer.Write([]byte("\n"))
		if err != nil {
			return err
		}
	}

	stanza, err := convertToStanza(data)
	if err != nil {
		return err
	}
	e.alreadyWritten = true

	_, err = stanza.WriteTo(e.writer)
	return err
}

func convertToStanza(data reflect.Value) (*Stanza, error) {
	if data.Type().Kind() != reflect.Struct {
		return nil, errors.New("can only Decode a Struct")
	}

	jsonData, err := json.Marshal(data.Interface())
	if err != nil {
		return nil, err
	}

	var paragraph Stanza
	if err := json.Unmarshal(jsonData, &paragraph); err != nil {
		return nil, err
	}

	return &paragraph, nil
}
