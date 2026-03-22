"""ytmusic_fetch.py — ytmusicapi를 사용하여 YouTube Music 좋아하는 노래를 가져오는 사이드카 스크립트.

사용법:
    echo '{"access_token":"...","refresh_token":"...","client_id":"...","client_secret":"..."}' | python3 ytmusic_fetch.py

stdin: JSON { access_token, refresh_token, client_id, client_secret }
stdout: JSON [{ external_id, title, artist, album_name, image_url }, ...]
stderr: 에러 메시지 (exit code 1)
"""

import json
import sys
import time

LIKED_SONGS_LIMIT = 500


def main():
    try:
        raw = sys.stdin.read()
        params = json.loads(raw)
    except (json.JSONDecodeError, ValueError) as e:
        print(f"stdin JSON 파싱 실패: {e}", file=sys.stderr)
        sys.exit(1)

    access_token = params.get("access_token", "")
    refresh_token = params.get("refresh_token", "")
    client_id = params.get("client_id", "")
    client_secret = params.get("client_secret", "")

    missing = [f for f in ("access_token", "refresh_token", "client_id", "client_secret")
               if not params.get(f)]
    if missing:
        print(f"필수 필드 누락: {', '.join(missing)}", file=sys.stderr)
        sys.exit(1)

    try:
        from ytmusicapi import YTMusic, OAuthCredentials
    except ImportError:
        print("ytmusicapi가 설치되지 않았습니다: pip install ytmusicapi", file=sys.stderr)
        sys.exit(1)

    try:
        oauth_credentials = OAuthCredentials(
            client_id=client_id,
            client_secret=client_secret,
        )
        token_json = json.dumps({
            "access_token": access_token,
            "refresh_token": refresh_token,
            "token_type": "Bearer",
            "scope": "https://www.googleapis.com/auth/youtube",
            "expires_at": int(time.time()) + 3600,
        })
        ytmusic = YTMusic(auth=token_json, oauth_credentials=oauth_credentials)
    except (ValueError, TypeError, KeyError) as e:
        print(f"YTMusic 초기화 실패: {e}", file=sys.stderr)
        sys.exit(1)

    try:
        liked = ytmusic.get_liked_songs(limit=LIKED_SONGS_LIMIT)
    except Exception as e:
        print(f"좋아하는 노래 가져오기 실패: {e}", file=sys.stderr)
        sys.exit(1)

    tracks = []
    for item in liked.get("tracks", []):
        video_id = item.get("videoId", "")
        if not video_id:
            continue

        title = item.get("title", "")

        artists = item.get("artists") or []
        artist = ", ".join(a.get("name", "") for a in artists if a.get("name"))

        album = item.get("album") or {}
        album_name = album.get("name", "")

        thumbnails = item.get("thumbnails") or []
        image_url = thumbnails[0].get("url", "") if thumbnails else ""

        tracks.append({
            "external_id": video_id,
            "title": title,
            "artist": artist,
            "album_name": album_name,
            "image_url": image_url,
        })

    json.dump(tracks, sys.stdout, ensure_ascii=False)


if __name__ == "__main__":
    main()
