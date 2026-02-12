package com.trah.abnotify

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.os.Build
import com.trah.abnotify.crypto.KeyManager
import com.trah.abnotify.data.AppDatabase

class AbnotifyApp : Application() {

    lateinit var database: AppDatabase
        private set

    lateinit var keyManager: KeyManager
        private set

    override fun onCreate() {
        super.onCreate()
        instance = this

        // Initialize database
        database = AppDatabase.getInstance(this)

        // Initialize key manager
        keyManager = KeyManager(this)
        keyManager.ensureKeysExist()

        // Create notification channels
        createNotificationChannels()
    }

    private fun createNotificationChannels() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

            // Service notification channel (minimum priority, can be hidden by user)
            val serviceChannel = NotificationChannel(
                CHANNEL_SERVICE,
                getString(R.string.notification_channel_name),
                NotificationManager.IMPORTANCE_MIN  // 最低优先级，尽量不打扰用户
            ).apply {
                description = getString(R.string.notification_channel_description)
                setShowBadge(false)
                lockscreenVisibility = android.app.Notification.VISIBILITY_SECRET  // 锁屏不显�?            }
            notificationManager.createNotificationChannel(serviceChannel)

            // Message notification channel (high priority)
            val messageChannel = NotificationChannel(
                CHANNEL_MESSAGES,
                "消息通知",
                NotificationManager.IMPORTANCE_MAX
            ).apply {
                description = "推送消息通知"
                enableVibration(true)
                enableLights(true)
                lockscreenVisibility = android.app.Notification.VISIBILITY_PUBLIC
            }
            notificationManager.createNotificationChannel(messageChannel)
        }
    }

    companion object {
        const val CHANNEL_SERVICE = "abnotify_service"
        const val CHANNEL_MESSAGES = "abnotify_messages"

        @Volatile
        private var instance: AbnotifyApp? = null

        fun getInstance(): AbnotifyApp {
            return instance ?: throw IllegalStateException("Application not initialized")
        }
    }
}
