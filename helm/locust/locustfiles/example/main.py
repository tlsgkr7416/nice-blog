from locust import HttpUser, task, between

class MyUser(HttpUser):
    # 사용자 간의 대기 시간 설정 (1초에서 5초 사이)
    wait_time = between(1, 5)

    @task
    def get_homepage(self):
        # GET 요청을 보내고 응답을 확인
        with self.client.get("/", catch_response=True) as response:
            if response.status_code == 200:
                response.success()  
            else:
                response.failure(f"Request failed with status code: {response.status_code}")
            
    @task
    def write_post(self):
        # POST 요청을 통해 게시물 작성
        self.client.post("/write", data={
            "title": "Sample Title",
            "author": "Author Name",
            "content": "This is the content of the post."
        })