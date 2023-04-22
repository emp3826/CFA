package com.github.kr328.clash.design.view

import android.content.Context
import android.graphics.drawable.Drawable
import android.util.AttributeSet
import androidx.annotation.AttrRes
import com.github.kr328.clash.design.R
import com.github.kr328.clash.design.databinding.ComponentLargeActionLabelBinding
import com.github.kr328.clash.design.util.*
import com.google.android.material.card.MaterialCardView

class LargeActionCard @JvmOverloads constructor(
    context: Context,
    attributeSet: AttributeSet? = null,
    @AttrRes defStyleAttr: Int = 0
) : MaterialCardView(context, attributeSet, defStyleAttr) {
    private val binding = ComponentLargeActionLabelBinding
        .inflate(context.layoutInflater, this, true)

    var text: CharSequence?
        get() = binding.textView.text
        set(value) {
            binding.textView.text = value
        }

    init {
        context.resolveClickableAttrs(attributeSet, defStyleAttr) {
            isFocusable = focusable(true)
            isClickable = clickable(true)
            foreground = foreground() ?: context.selectableItemBackground
        }

        context.theme.obtainStyledAttributes(
            attributeSet,
            R.styleable.LargeActionCard,
            defStyleAttr,
            0
        ).apply {
            try {
                text = getString(R.styleable.LargeActionCard_text)
            } finally {
                recycle()
            }
        }

        minimumHeight = context.getPixels(R.dimen.large_action_card_min_height)
        radius = context.getPixels(R.dimen.large_action_card_radius).toFloat()
        elevation = context.getPixels(R.dimen.large_action_card_elevation).toFloat()
        setCardBackgroundColor(context.resolveThemedColor(R.attr.colorSurface))
    }
}
