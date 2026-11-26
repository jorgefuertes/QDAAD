{ Stub. The real unit is not in the checkout: work/additional_sources has only
  SCRMAKER.PAS and PLUS3.ASM. It is needed to link, not to run — nothing on the
  pc target calls it, and anything that does says so instead of writing wrong
  bytes. }
unit SimpleRLEBlockCompressor;
{$mode objfpc}{$H+}

interface

uses sysutils;

type TByteBuf8000 = array [0..32767] of byte;

function CompressBuffer(DataBuffer: TByteBuf8000; Size: word; BaseAddr: word = 0): TBytes;

implementation

function CompressBuffer(DataBuffer: TByteBuf8000; Size: word; BaseAddr: word = 0): TBytes;
begin
  WriteLn('SimpleRLEBlockCompressor is a stub in this build: only the pc target works.');
  Halt(3);
  CompressBuffer := nil;
end;

end.
