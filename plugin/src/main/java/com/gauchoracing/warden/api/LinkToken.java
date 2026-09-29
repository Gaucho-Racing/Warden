package com.gauchoracing.warden.api;

/** A pending account link. The service builds the URL so the portal can move without a new jar. */
public record LinkToken(String token, String url, String expiresAt) {}
