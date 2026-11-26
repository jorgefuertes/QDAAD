{ Stub. See SimpleRLEBlockCompressor.pas. }
unit ZX0Compressor;
{$mode objfpc}{$H+}

interface

type TZX0Bytes = array of byte;

function ZX0Compress(const RawData; RawSize: longint): TZX0Bytes;

implementation

function ZX0Compress(const RawData; RawSize: longint): TZX0Bytes;
begin
  WriteLn('ZX0Compressor is a stub in this build: only the pc target works.');
  Halt(3);
  ZX0Compress := nil;
end;

end.
