def escape():
    s = """ROBLOX	com.roblox.client
Netflix	com.netflix.mediaclient
YouTube Kids	com.google.android.apps.youtube.kids
Facebook Messenger	com.facebook.orca
Free Fire	com.dts.freefireth
Gacha Club	air.com.lunime.gachaclub
Messenger Kids	com.facebook.talk
Among Us!	com.innersloth.spacemafia
Gacha Life	air.com.lunime.gachalife
Tubi TV	com.tubitv
Google Classroom	com.google.android.apps.classroom
Candy Crush Soda Saga	com.king.candycrushsodasaga
Google Photos	com.google.android.apps.photos
Homescapes	com.playrix.homescapes
Hangouts	com.google.android.talk
June's Journey	net.wooga.junes_journey_hidden_object_mystery_game
Gmail	com.google.android.gm
YouTube Music	com.google.android.apps.youtube.music
Viki	com.viki.android"""
    return s.replace("\n", "\\n").replace("\t", "\\t")

if __name__ == "__main__":
    print(escape())