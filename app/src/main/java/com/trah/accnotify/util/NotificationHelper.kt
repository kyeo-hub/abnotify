package com.trah.accnotify.util

import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.core.app.NotificationCompat
import com.trah.accnotify.AccnotifyApp
import com.trah.accnotify.R
import com.trah.accnotify.ui.MainActivity
import java.util.concurrent.atomic.AtomicInteger

object NotificationHelper {

    private val notificationIdCounter = AtomicInteger(100)

    fun showNotification(
        context: Context,
        messageId: String,
        title: String,
        body: String,
        group: String? = null,
        url: String? = null
    ) {
        val notificationManager = AccnotifyApp.getInstance().getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        val notificationId = notificationIdCounter.getAndIncrement()

        // Content intent - open app or URL
        val contentIntent = if (!url.isNullOrEmpty()) {
            Intent(Intent.ACTION_VIEW, Uri.parse(url))
        } else {
            Intent(context, MainActivity::class.java)
        }

        val pendingIntent = PendingIntent.getActivity(
            context,
            notificationId,
            contentIntent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        )

        // Use group parameter if provided, otherwise use messageId as group key
        val groupKey = group ?: "accnotify_$messageId"

        val builder = NotificationCompat.Builder(context, AccnotifyApp.CHANNEL_MESSAGES)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setStyle(NotificationCompat.BigTextStyle().bigText(body))
            .setPriority(NotificationCompat.PRIORITY_MAX)
            .setDefaults(NotificationCompat.DEFAULT_ALL)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .setVibrate(longArrayOf(0, 500, 200, 500))
            .setGroup(groupKey)

        notificationManager.notify(messageId, notificationId, builder.build())
    }
}
