# 测试多层嵌套的Python代码，用于验证认知复杂度计算

def test_deep_nesting(data, config):
    """测试三层嵌套的认知复杂度"""
    result = []
    
    # 第1层嵌套: if
    if data is not None:
        # 第2层嵌套: for
        for item in data:
            if item is not None:
                # 第3层嵌套: if
                if item.get('active', False):
                    if config.get('validate', False):
                        # 第4层嵌套
                        if item.get('score', 0) > 10:
                            result.append(item)
    
    # 另一个分支
    if config.get('sort', False):
        if len(result) > 0:
            result.sort(key=lambda x: x.get('id', 0))
    
    return result


class OrderProcessor:
    """测试类的方法"""
    
    def __init__(self, order_service, payment_service, inventory_service, logger, config):
        self.order_service = order_service
        self.payment_service = payment_service
        self.inventory_service = inventory_service
        self.logger = logger
        self.config = config
    
    def process_complex_order(self, order, user, options, payment_info, shipping_info):
        """长参数 + 多层嵌套"""
        success = False
        
        if order is not None:
            if user is not None:
                if user.get('active', False):
                    if order.get('items', []):
                        for item in order['items']:
                            if item.get('quantity', 0) > 0:
                                if self.inventory_service.check_stock(item['id']):
                                    success = True
                                else:
                                    self.logger.warn("库存不足")
                    else:
                        raise ValueError("订单为空")
                else:
                    raise ValueError("用户未激活")
            else:
                raise ValueError("用户不存在")
        else:
            raise ValueError("订单不存在")
        
        return success
