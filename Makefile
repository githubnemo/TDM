# Copyright 2009 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

include $(GOROOT)/src/Make.inc

TARG=tdm
GOFILES=\
	tdm.go\
	source.go\
	network.go\
	packet.go\
	sink.go

include $(GOROOT)/src/Make.cmd
