package com.gauchoracing.warden;

/** Log-friendly descriptions of exceptions. */
public final class Errors {

    private Errors() {}

    /**
     * Describes a failure for a log line. HttpClient wraps transport errors in
     * exceptions with a null message (a refused connection surfaces as a bare
     * ConnectException), so fall back to the type and the first cause that
     * actually says something.
     */
    public static String describe(Throwable e) {
        String type = e.getClass().getSimpleName();
        for (Throwable t = e; t != null; t = t.getCause()) {
            String message = t.getMessage();
            if (message != null && !message.isBlank()) {
                return t == e ? message : type + ": " + message;
            }
        }
        return type;
    }
}
