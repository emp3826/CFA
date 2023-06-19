package com.github.kr328.clash.service.remote

import com.github.kr328.clash.service.model.Profile
import com.github.kr328.kaidl.BinderInterface
import java.util.UUID

@BinderInterface
interface IProfileManager {
    suspend fun create(type: Profile.Type, name: String, source: String = ""): UUID
    suspend fun commit(uuid: UUID, callback: IFetchObserver? = null)
    suspend fun release(uuid: UUID)
    suspend fun delete(uuid: UUID)
    suspend fun patch(uuid: UUID, name: String, source: String)
    suspend fun queryByUUID(uuid: UUID): Profile?
    suspend fun queryAll(): List<Profile>
    suspend fun setActive(profile: Profile)
}
