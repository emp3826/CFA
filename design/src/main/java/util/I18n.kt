package com.github.kr328.clash.design.util

import android.content.Context
import com.github.kr328.clash.common.compat.preferredLocale
import com.github.kr328.clash.core.model.Provider
import com.github.kr328.clash.design.R
import com.github.kr328.clash.service.model.Profile
import java.text.SimpleDateFormat
import java.util.*

fun Profile.Type.toString(context: Context): String {
    return when (this) {
        Profile.Type.File -> context.getString(R.string.file)
        Profile.Type.Url -> context.getString(R.string.url)
        Profile.Type.External -> context.getString(R.string.external)
    }
}

fun Provider.type(context: Context): String {
    val type = context.getString(R.string.proxy)
    return context.getString(R.string.format_provider_type, type)
}

fun Long.toBytesString(): String {
    return when {
        this > 1024 * 1024 * 1024 ->
            String.format("%.2f GiB", (this.toDouble() / 1024 / 1024 / 1024))
        this > 1024 * 1024 ->
            String.format("%.2f MiB", (this.toDouble() / 1024 / 1024))
        this > 1024 ->
            String.format("%.2f KiB", (this.toDouble() / 1024))
        else ->
            "$this Bytes"
    }
}
