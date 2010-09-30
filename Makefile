include $(GOROOT)/src/Make.inc

TARG=mcs
GOFILES=\
	mcs.go \
	proto.go

include $(GOROOT)/src/Make.cmd
