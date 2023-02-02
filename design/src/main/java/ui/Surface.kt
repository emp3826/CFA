package com.github.kr328.clash.design.ui

import androidx.databinding.BaseObservable
import androidx.databinding.Bindable

class Surface : BaseObservable() {
    var insets: Insets = Insets.EMPTY
        set(value) {
            field = value
        }
}
