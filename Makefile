include $(GOROOT)/src/Make.inc

# TODO Properly build and link packages
GC += -I nbt/_obj
LD += -L nbt/_obj

TARG=mcs
GOFILES=\
	mcs.go \
	proto.go \
	chunk.go \
	game.go

include $(GOROOT)/src/Make.cmd
