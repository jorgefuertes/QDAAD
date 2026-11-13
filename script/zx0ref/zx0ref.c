/*
 * Compresses a file with Einar Saukas' own ZX0 coder, so that ours has
 * something to be held against.
 *
 * None of his code is here. optimize() and compress() are compiled straight
 * out of work/ZXDAAD128/DCP, which is where the coder the reference pipeline
 * runs lives — zx0.exe is an x86 binary, but its source ships beside the
 * interpreter. This is only the main() that tree does not have: DCP.c wraps
 * the coder in screen cutting and sector arithmetic, and what the comparison
 * wants is the stream and nothing else.
 *
 * It is a tool of this repository and not a part of the compiler. Nothing in
 * internal/ knows it exists.
 *
 *	zx0ref <invert:0|1> <in> <out>
 *
 * invert picks the format. DCP.c:301 passes TRUE, which is ZX0's classic
 * coding and the one dzx0_standard.asm reads — see compare-zx0.sh, which
 * checks that first and refuses to go on if it is ever not so.
 */
#include <stdio.h>
#include <stdlib.h>

#include "zx0.h"

/* As DCP.c has it: 255 high values of 128 positions each. */
#define MAX_OFFSET_ZX0 32640

int main(int argc, char *argv[]) {
	if (argc != 4) {
		fprintf(stderr, "usage: %s <invert:0|1> <in> <out>\n", argv[0]);
		return EXIT_FAILURE;
	}

	int invert = atoi(argv[1]);

	FILE *in = fopen(argv[2], "rb");
	if (!in) {
		fprintf(stderr, "zx0ref: cannot read %s\n", argv[2]);
		return EXIT_FAILURE;
	}

	if (fseek(in, 0, SEEK_END) != 0) {
		fprintf(stderr, "zx0ref: cannot measure %s\n", argv[2]);
		return EXIT_FAILURE;
	}

	long size = ftell(in);
	if (size <= 0) {
		fprintf(stderr, "zx0ref: %s holds nothing to compress\n", argv[2]);
		return EXIT_FAILURE;
	}

	rewind(in);

	unsigned char *data = malloc(size);
	if (!data) {
		fprintf(stderr, "zx0ref: no room for %ld bytes\n", size);
		return EXIT_FAILURE;
	}

	if (fread(data, 1, size, in) != (size_t)size) {
		fprintf(stderr, "zx0ref: short read on %s\n", argv[2]);
		return EXIT_FAILURE;
	}

	fclose(in);

	int out_size, delta;
	unsigned char *out = compress(
		optimize(data, size, 0, MAX_OFFSET_ZX0),
		data, size, 0, FALSE, invert, &out_size, &delta);

	FILE *of = fopen(argv[3], "wb");
	if (!of) {
		fprintf(stderr, "zx0ref: cannot write %s\n", argv[3]);
		return EXIT_FAILURE;
	}

	if (fwrite(out, 1, out_size, of) != (size_t)out_size) {
		fprintf(stderr, "zx0ref: cannot write %s\n", argv[3]);
		return EXIT_FAILURE;
	}

	fclose(of);

	return EXIT_SUCCESS;
}
