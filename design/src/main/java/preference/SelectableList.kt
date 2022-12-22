package com.github.kr328.clash.design.preference

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.appcompat.widget.ListPopupWindow
import com.github.kr328.clash.design.R
import com.github.kr328.clash.design.util.getPixels
import com.github.kr328.clash.design.util.measureWidth
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlin.reflect.KMutableProperty0

interface SelectableListPreference<T> : ClickablePreference {
    var selected: Int

    var listener: OnChangedListener?
}

fun <T> PreferenceScreen.selectableList(
    value: KMutableProperty0<T>,
    values: Array<T>,
    valuesText: Array<Int>,
    @StringRes title: Int,
    @DrawableRes icon: Int? = null,
    configure: SelectableListPreference<T>.() -> Unit = {},
): SelectableListPreference<T> {
    val impl = object : SelectableListPreference<T>, ClickablePreference by clickable(title, icon) {
        override var selected: Int = 0
            set(value) {
                field = value

                this.summary = context.getText(valuesText[value])
            }
        override var listener: OnChangedListener? = null
    }

    impl.configure()

    launch(Dispatchers.Main) {
        val initial = withContext(Dispatchers.IO) {
            value.get()
        }

        impl.selected = values.indexOf(initial)
    }

    return impl
}
