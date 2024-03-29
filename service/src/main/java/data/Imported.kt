package com.github.kr328.clash.service.data

import androidx.room.ColumnInfo
import androidx.room.Entity
import androidx.room.TypeConverters
import com.github.kr328.clash.service.model.Profile
import java.util.UUID

@Entity(tableName = "imported", primaryKeys = ["uuid"])
@TypeConverters(Converters::class)
data class Imported(
    @ColumnInfo(name = "uuid") val uuid: UUID,
    @ColumnInfo(name = "name") val name: String,
    @ColumnInfo(name = "type") val type: Profile.Type,
    @ColumnInfo(name = "source") val source: String,
    @ColumnInfo(name = "createdAt") val createdAt: Long,
)
