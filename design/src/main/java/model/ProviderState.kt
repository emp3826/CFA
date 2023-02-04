package com.github.kr328.clash.design.model

import androidx.databinding.BaseObservable
import com.github.kr328.clash.core.model.Provider

class ProviderState(
    val provider: Provider,
    updatedAt: Long,
    updating: Boolean,
) : BaseObservable() {
    var updatedAt: Long = updatedAt
        set(value) {
            field = value
        }

    var updating: Boolean = updating
        set(value) {
            field = value
        }
}
