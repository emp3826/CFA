@file:Suppress("DEPRECATION")

package com.github.kr328.clash.common.compat

import android.text.Html
import android.text.Spanned

fun fromHtmlCompat(content: String): Spanned {
    return Html.fromHtml(content, Html.FROM_HTML_MODE_COMPACT)
}
