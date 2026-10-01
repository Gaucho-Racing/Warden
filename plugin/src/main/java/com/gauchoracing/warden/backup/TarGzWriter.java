package com.gauchoracing.warden.backup;

import java.io.Closeable;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.zip.Deflater;
import java.util.zip.GZIPOutputStream;

/**
 * Writes a gzipped POSIX tar.
 *
 * <p>Hand-rolled because the plugin ships no dependencies — everything it
 * compiles against is {@code provided} by the server, which is what keeps
 * the jar at a few KB and means no shade plugin. Pulling in
 * commons-compress for one archive format would change that for the whole
 * build.
 *
 * <p>The format is ustar with PAX extended headers, which are emitted only
 * when a path will not fit the 100-byte name field or a file exceeds the 8
 * GiB the octal size field can express. Both are rare in a Minecraft data
 * directory and both silently corrupt an archive if ignored.
 */
public final class TarGzWriter implements Closeable {

    private static final int BLOCK = 512;
    /** The largest size representable in the 12-byte octal size field. */
    private static final long MAX_OCTAL_SIZE = (1L << 33) - 1;

    private static final byte[] PADDING = new byte[BLOCK];

    private final OutputStream out;
    private final byte[] header = new byte[BLOCK];
    private final byte[] copyBuffer = new byte[64 * 1024];
    private long paxSequence;

    public TarGzWriter(OutputStream sink) throws IOException {
        // Region files hold zlib-compressed chunks already, so the slow
        // deflate levels buy a percent or two for several times the CPU.
        // On a live server that CPU is competing with the tick loop.
        this.out = new GZIPOutputStream(sink, 64 * 1024) {
            {
                def.setLevel(Deflater.BEST_SPEED);
            }
        };
    }

    public void addDirectory(String name, long modifiedSeconds) throws IOException {
        if (!name.endsWith("/")) {
            name = name + "/";
        }
        writeEntry(name, '5', 0L, modifiedSeconds, 0755);
    }

    /**
     * Writes a file entry of exactly {@code size} bytes.
     *
     * <p>The size went into the header before the first byte was read, so a
     * file the server rewrites mid-archive must not be allowed to change the
     * length: a short read is zero-padded and a long one truncated. Either
     * costs one corrupt file; disagreeing with the header costs the archive.
     */
    public void addFile(String name, Path source, long size, long modifiedSeconds) throws IOException {
        writeEntry(name, '0', size, modifiedSeconds, 0644);
        long remaining = size;
        try (InputStream in = Files.newInputStream(source)) {
            while (remaining > 0) {
                int wanted = (int) Math.min(copyBuffer.length, remaining);
                int read = in.read(copyBuffer, 0, wanted);
                if (read < 0) {
                    break;
                }
                out.write(copyBuffer, 0, read);
                remaining -= read;
            }
        }
        while (remaining > 0) {
            int chunk = (int) Math.min(PADDING.length, remaining);
            out.write(PADDING, 0, chunk);
            remaining -= chunk;
        }
        pad(size);
    }

    /** Two zero blocks mark the end of the archive; gzip is finished by close. */
    @Override
    public void close() throws IOException {
        out.write(PADDING);
        out.write(PADDING);
        out.close();
    }

    private void writeEntry(String name, char type, long size, long modifiedSeconds, int mode)
            throws IOException {
        byte[] nameBytes = name.getBytes(StandardCharsets.UTF_8);
        boolean longName = nameBytes.length > 100;
        boolean hugeFile = size > MAX_OCTAL_SIZE;
        if (longName || hugeFile) {
            writePaxHeader(name, longName, size, hugeFile, modifiedSeconds);
        }

        java.util.Arrays.fill(header, (byte) 0);
        // With a PAX header in front, the ustar fields are a fallback for
        // readers that ignore it; truncating here is correct, not lossy.
        putString(0, 100, longName ? truncateUtf8(nameBytes, 100) : name);
        putOctal(100, 8, mode);
        putOctal(108, 8, 0);
        putOctal(116, 8, 0);
        putOctal(124, 12, hugeFile ? 0 : size);
        putOctal(136, 12, modifiedSeconds);
        header[156] = (byte) type;
        putString(257, 6, "ustar");
        header[263] = '0';
        header[264] = '0';
        putString(265, 32, "root");
        putString(297, 32, "root");
        writeChecksum();
        out.write(header);
    }

    /**
     * A PAX extended header is itself a tar entry whose body is a list of
     * {@code "<length> key=value\n"} records, where the length counts itself.
     * That self-reference is why the length is solved by iteration.
     */
    private void writePaxHeader(String name, boolean longName, long size, boolean hugeFile,
            long modifiedSeconds) throws IOException {
        StringBuilder records = new StringBuilder();
        if (longName) {
            records.append(paxRecord("path", name));
        }
        if (hugeFile) {
            records.append(paxRecord("size", Long.toString(size)));
        }
        byte[] body = records.toString().getBytes(StandardCharsets.UTF_8);

        java.util.Arrays.fill(header, (byte) 0);
        putString(0, 100, "PaxHeaders/" + (++paxSequence));
        putOctal(100, 8, 0644);
        putOctal(108, 8, 0);
        putOctal(116, 8, 0);
        putOctal(124, 12, body.length);
        putOctal(136, 12, modifiedSeconds);
        header[156] = 'x';
        putString(257, 6, "ustar");
        header[263] = '0';
        header[264] = '0';
        writeChecksum();
        out.write(header);
        out.write(body);
        pad(body.length);
    }

    private static String paxRecord(String key, String value) {
        int withoutLength = key.getBytes(StandardCharsets.UTF_8).length
                + value.getBytes(StandardCharsets.UTF_8).length
                + 3; // space, '=', '\n'
        int length = withoutLength + Integer.toString(withoutLength).length();
        // Adding the digits can push the total across a power of ten, which
        // adds another digit. Two rounds always settle it at these sizes.
        length = withoutLength + Integer.toString(length).length();
        return length + " " + key + "=" + value + "\n";
    }

    private void pad(long written) throws IOException {
        int remainder = (int) (written % BLOCK);
        if (remainder != 0) {
            out.write(PADDING, 0, BLOCK - remainder);
        }
    }

    /**
     * The checksum is the unsigned sum of every header byte with its own
     * field read as eight spaces, stored as six octal digits, a NUL and a
     * space.
     */
    private void writeChecksum() {
        java.util.Arrays.fill(header, 148, 156, (byte) ' ');
        int sum = 0;
        for (byte b : header) {
            sum += b & 0xFF;
        }
        putOctal(148, 7, sum);
        header[154] = 0;
        header[155] = ' ';
    }

    private void putString(int offset, int length, String value) {
        byte[] bytes = value.getBytes(StandardCharsets.UTF_8);
        System.arraycopy(bytes, 0, header, offset, Math.min(bytes.length, length));
    }

    /** Right-aligned octal, zero-padded, NUL-terminated — the ustar convention. */
    private void putOctal(int offset, int length, long value) {
        String octal = Long.toOctalString(value);
        int digits = length - 1;
        if (octal.length() > digits) {
            octal = octal.substring(octal.length() - digits);
        }
        int start = offset + digits - octal.length();
        for (int i = offset; i < start; i++) {
            header[i] = '0';
        }
        byte[] bytes = octal.getBytes(StandardCharsets.US_ASCII);
        System.arraycopy(bytes, 0, header, start, bytes.length);
        header[offset + digits] = 0;
    }

    /** Trims to a byte budget without splitting a multi-byte character. */
    private static String truncateUtf8(byte[] bytes, int limit) {
        int end = limit;
        while (end > 0 && (bytes[end] & 0xC0) == 0x80) {
            end--;
        }
        return new String(bytes, 0, end, StandardCharsets.UTF_8);
    }
}
