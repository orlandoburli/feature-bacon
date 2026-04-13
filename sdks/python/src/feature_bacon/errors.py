class BaconError(Exception):
    def __init__(
        self,
        status_code: int,
        type_: str,
        title: str,
        detail: str,
        instance: str,
    ):
        self.status_code = status_code
        self.type = type_
        self.title = title
        self.detail = detail
        self.instance = instance
        super().__init__(f"{title} ({status_code}): {detail}")
