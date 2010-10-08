include $(GOROOT)/src/Make.inc

# TODO Properly build and link packages
GC += -I nbt/_obj
LD += -L nbt/_obj

TARG=chunkymonkey
GOFILES=\
	chunkymonkey.go \
	proto.go \
	chunk.go \
	game.go \
	player.go \
	entity.go \
	record.go \

include $(GOROOT)/src/Make.cmd
