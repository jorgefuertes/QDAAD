# Two stub units, so that SCRMAKER will build

`work/additional_sources/SCRMAKER.PAS` is the reference image converter, and it
is the source that settles the `.MSD` format of the PC target and the `.128` of
the +3 and the 128K. It does not build on its own: its `USES` line names
`SimpleRLEBlockCompressor` and `ZX0Compressor`, and neither unit was published
with it.

These two are stubs that satisfy the linker and nothing else. Both halt with a
message if anything calls them, which is deliberate: a stub that returned empty
bytes would make SCRMAKER write a file that looks like a conversion and is not.

Nothing on the `pc` target reaches them — `CompressBuffer` is the C64's and the
Amstrad's run encoder, `ZX0Compress` is the SAM Coupé's — so
[compare-scrmaker.sh](../compare-scrmaker.sh) builds SCRMAKER with these and
uses it for `pc` only.

If the real units ever turn up, delete these and the harness keeps working: it
compiles whatever is beside the source.
